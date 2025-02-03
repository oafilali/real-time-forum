package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

func checkEmailExists(w http.ResponseWriter, email string) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	if errorCheckHandlers(w, "Database error", err, http.StatusInternalServerError) {
		return true
	}
	if count > 1 {
		http.Error(w, "Database integrity error", http.StatusInternalServerError)
		return true
	} else if count == 1 {
		http.Error(w, "Email already taken", http.StatusBadRequest)
		return true
	}
	return false
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

func saveUser(username, email, hashedPassword string) error {
	_, err := db.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, hashedPassword,
	)
	return err
}

func errorCheckHandlers(w http.ResponseWriter, msg string, err error, code int) bool {
	if err != nil {
		http.Error(w, msg, code)
		log.Println(msg, err)
		return true
	}
	return false
}

func authenticateUser(email, password string) (int, error) {
	var userID int
	var storedHash string
	err := db.QueryRow(
		"SELECT id, password FROM users WHERE email = ?",
		email,
	).Scan(&userID, &storedHash)
	if err != nil {
		return 0, err
	}
	if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) != nil {
		return 0, fmt.Errorf("invalid password")
	}
	return userID, nil
}

func createSession(w http.ResponseWriter, userID int) error {
	sessionID, err := uuid.NewV4()
	if err != nil {
		return err
	}

	expires := time.Now().Add(2 * time.Hour)
	_, err = db.Exec(
		"INSERT INTO sessions (session_id, id, expires_at) VALUES (?, ?, ?)",
		sessionID.String(), userID, expires,
	)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID.String(),
		Expires:  expires,
		HttpOnly: true,
	})
	return nil
}

func deleteSession(sessionID string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE session_id = ?", sessionID)
	return err
}

func getUserIDFromSession(r *http.Request) (int, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, err
	}

	var userID int
	err = db.QueryRow(
		"SELECT id FROM sessions WHERE session_id = ? AND expires_at > CURRENT_TIMESTAMP",
		cookie.Value,
	).Scan(&userID)
	return userID, err
}

func createPost(userID int, title, content, category string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert post into Posts table with category included
	_, err = tx.Exec(
		"INSERT INTO posts (user_id, title, content, category) VALUES (?, ?, ?, ?)",
		userID, title, content, category,
	)
	if err != nil {
		return err
	}

	// Commit the transaction if everything is successful
	return tx.Commit()
}

func fetchReactionsNumber(itemID int, isComment bool) (likes, dislikes int, err error) {
	var query string
	if isComment {
		query = `
			SELECT 
				COUNT(CASE WHEN type = 'like' THEN 1 END) AS likes,
				COUNT(CASE WHEN type = 'dislike' THEN 1 END) AS dislikes
			FROM reactions 
			WHERE comment_id = ?`
	} else {
		query = `
			SELECT 
				COUNT(CASE WHEN type = 'like' THEN 1 END) AS likes,
				COUNT(CASE WHEN type = 'dislike' THEN 1 END) AS dislikes
			FROM reactions 
			WHERE post_id = ?`
	}

	err = db.QueryRow(query, itemID).Scan(&likes, &dislikes)
	return
}

func fetchPosts() ([]posts, error) {
	postRows, err := db.Query(`SELECT 
    	p.id, 
    	p.title, 
    	COALESCE(SUM(CASE WHEN r.type = 'like' THEN 1 ELSE 0 END), 0) AS likes,
    	COALESCE(SUM(CASE WHEN r.type = 'dislike' THEN 1 ELSE 0 END), 0) AS dislikes
	FROM posts p
	LEFT JOIN reactions r ON p.id = r.post_id AND r.comment_id IS NULL
	GROUP BY p.id, p.title;`)
	if err != nil {
		return nil, err
	}
	defer postRows.Close()

	var allPosts []posts

	// Iterate over each post
	for postRows.Next() {
		var post posts

		// Scan the post data
		err := postRows.Scan(&post.ID, &post.Title, &post.Likes, &post.Dislikes)
		if err != nil {
			return nil, err
		}
		allPosts = append(allPosts, post)
	}
	return allPosts, nil
}

func getPostId() (id int, err error) {
	err = db.QueryRow("SELECT last_insert_rowid()").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func fetchPost(postID string) (Post, error) {
	var post Post
	err := db.QueryRow("SELECT id, user_id, title, content, category FROM posts WHERE id = ?", postID).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.Category)
	if err != nil {
		return Post{}, err
	}
	username := ""
	err = db.QueryRow("SELECT username FROM users WHERE id= ?", post.UserID).Scan(&username)
	if err != nil {
		return Post{}, err
	}
	post.Username = username
	return post, err
}

func fetchCommentsForPost(postID int) ([]comment, error) {
	// Query the comments for the specific post
	commentRows, err := db.Query(`
    SELECT c.id, c.user_id, c.content, u.username
    FROM comments c
    JOIN users u ON u.id = c.user_id
    WHERE c.post_id = ?`, postID)
	if err != nil {
		return nil, err
	}
	defer commentRows.Close()

	var comments []comment

	// Iterate over comments for this post
	for commentRows.Next() {
		var comment comment
		err := commentRows.Scan(&comment.ID, &comment.UserID, &comment.Content, &comment.Username)
		if err != nil {
			return nil, err
		}

		// Fetch likes and dislikes for each comment
		comment.Likes, comment.Dislikes, err = fetchReactionsNumber(comment.ID, true)
		if err != nil {
			return nil, err
		}

		comments = append(comments, comment)
	}

	// Check if there was any issue after iterating over rows
	if err := commentRows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}
