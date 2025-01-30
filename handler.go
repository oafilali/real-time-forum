package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// sessions stores active user sessions
var sessions = make(map[string]int) // session ID -> user ID

// registerHandler handles user registration
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Check if email already exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "Email already taken", http.StatusBadRequest)
			return
		}

		// Insert new user
		_, err = db.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)", username, email, password)
		if err != nil {
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "User registered successfully")
	} else {
		http.ServeFile(w, r, "./html/register.html")
	}
}

// loginHandler handles user login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		email := r.FormValue("email")
		password := r.FormValue("password")

		var dbPassword string
		var userID int
		err := db.QueryRow("SELECT id, password FROM users WHERE email = ?", email).Scan(&userID, &dbPassword)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if password != dbPassword {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		// Create a new session
		sessionID := uuid.New().String()
		sessions[sessionID] = userID

		// Set the session ID in a cookie
		http.SetCookie(w, &http.Cookie{
			Name:    "session_id",
			Value:   sessionID,
			Expires: time.Now().Add(24 * time.Hour), // Session expires in 24 hours
		})

		http.Redirect(w, r, "/posts", http.StatusFound)
	} else {
		http.ServeFile(w, r, "./html/login.html")
	}
}

// postHandler handles creating a new post
func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Get the session ID from the cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Not logged in", http.StatusUnauthorized)
			return
		}

		// Get the user ID from the session
		userID, ok := sessions[cookie.Value]
		if !ok {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		title := r.FormValue("title")
		content := r.FormValue("content")
		category := r.FormValue("category")

		// Insert the post into the database
		_, err = db.Exec("INSERT INTO posts (user_id, title, content, category) VALUES (?, ?, ?, ?)", userID, title, content, category)
		if err != nil {
			http.Error(w, "Failed to create post", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Post created successfully")
	} else {
		http.ServeFile(w, r, "./html/post.html")
	}
}

// postsHandler displays all posts
func postsHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch all posts
	postRows, err := db.Query("SELECT id, user_id, title, content, category FROM posts")
	if err != nil {
		http.Error(w, "Failed to fetch posts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer postRows.Close()

	var posts []struct {
		ID       int
		UserID   int
		Title    string
		Content  string
		Category string
		Likes    int
		Dislikes int
		Comments []struct {
			ID      int
			UserID  int
			Content string
		}
	}

	// Iterate over each post
	for postRows.Next() {
		var post struct {
			ID       int
			UserID   int
			Title    string
			Content  string
			Category string
			Likes    int
			Dislikes int
			Comments []struct {
				ID      int
				UserID  int
				Content string
			}
		}

		// Scan the post data
		err := postRows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Category)
		if err != nil {
			http.Error(w, "Failed to scan post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Fetch the number of likes for this post
		var likes int
		err = db.QueryRow("SELECT COUNT(*) FROM reactions WHERE post_id = ? AND type = 'like'", post.ID).Scan(&likes)
		if err != nil {
			http.Error(w, "Failed to fetch likes: "+err.Error(), http.StatusInternalServerError)
			return
		}
		post.Likes = likes

		// Fetch the number of dislikes for this post
		var dislikes int
		err = db.QueryRow("SELECT COUNT(*) FROM reactions WHERE post_id = ? AND type = 'dislike'", post.ID).Scan(&dislikes)
		if err != nil {
			http.Error(w, "Failed to fetch dislikes: "+err.Error(), http.StatusInternalServerError)
			return
		}
		post.Dislikes = dislikes

		// Fetch comments for this post
		commentRows, err := db.Query("SELECT id, user_id, content FROM comments WHERE post_id = ?", post.ID)
		if err != nil {
			http.Error(w, "Failed to fetch comments: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer commentRows.Close()

		// Iterate over comments for this post
		for commentRows.Next() {
			var comment struct {
				ID      int
				UserID  int
				Content string
			}
			err := commentRows.Scan(&comment.ID, &comment.UserID, &comment.Content)
			if err != nil {
				http.Error(w, "Failed to scan comment: "+err.Error(), http.StatusInternalServerError)
				return
			}
			post.Comments = append(post.Comments, comment)
		}

		// Add the post (with likes, dislikes, and comments) to the posts slice
		posts = append(posts, post)
	}

	// Parse the template
	tmpl, err := template.ParseFiles("./html/posts.html")
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	// Execute the template with the posts data
	err = tmpl.Execute(w, posts)
	if err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		return
	}
}

// commentHandler handles adding a comment to a post
func commentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Get the session ID from the cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Not logged in", http.StatusUnauthorized)
			return
		}

		// Get the user ID from the session
		userID, ok := sessions[cookie.Value]
		if !ok {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		postID := r.FormValue("post_id")
		content := r.FormValue("content")

		// Log the values for debugging
		log.Printf("Adding comment: post_id=%s, user_id=%d, content=%s\n", postID, userID, content)

		// Insert the comment into the database
		_, err = db.Exec("INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)", postID, userID, content)
		if err != nil {
			log.Println("Failed to add comment:", err)
			http.Error(w, "Failed to add comment", http.StatusInternalServerError)
			return
		}

		// Redirect back to the posts page
		http.Redirect(w, r, "/posts", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

// likeHandler handles liking or disliking a post
func likeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Get the session ID from the cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Not logged in", http.StatusUnauthorized)
			return
		}

		// Get the user ID from the session
		userID, ok := sessions[cookie.Value]
		if !ok {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		postID := r.FormValue("post_id")
		reactionType := r.FormValue("type")
		var currentReact string

		res, _ := db.Query(`SELECT type FROM reactions WHERE user_id = ? AND post_id = ?;`, userID, postID)
		if !res.Next() {
			res.Close()
			fmt.Printf("Adding %s to post %v from user %v\n", reactionType, postID, userID)
			// Insert the reaction into the database
			_, err = db.Exec("INSERT INTO reactions (post_id, user_id, type) VALUES (?, ?, ?)", postID, userID, reactionType)
			if err != nil {
				http.Error(w, "Failed to react to post", http.StatusInternalServerError)
				return
			}
		} else {
			err := res.Scan(&currentReact)
			res.Close()
			if err != nil {
				http.Error(w, "Failed to fetch reaction", http.StatusInternalServerError)
				return
			}
			if (reactionType == "like" && currentReact == "dislike") || (reactionType == "dislike" && currentReact == "like") {
				_, err = db.Exec(`DELETE FROM reactions WHERE user_id = ? AND post_id = ?;`, userID, postID)

				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				_, err = db.Exec("INSERT INTO reactions (post_id, user_id, type) VALUES (?, ?, ?)", postID, userID, reactionType)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		// Redirect back to the posts page
		http.Redirect(w, r, "/posts#"+postID, http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

// logoutHandler handles user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get the session ID from the cookie
	cookie, err := r.Cookie("session_id")
	if err != nil {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	// Delete the session
	delete(sessions, cookie.Value)

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_id",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour), // Expire the cookie
	})

	fmt.Fprintf(w, "Logout successful")
}

// filterHandler handles filtering posts by category
func filterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		category := r.URL.Query().Get("category")

		rows, err := db.Query("SELECT id, user_id, title, content, category FROM posts WHERE category = ?", category)
		if err != nil {
			http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var posts []struct {
			ID       int
			UserID   int
			Title    string
			Content  string
			Category string
		}

		for rows.Next() {
			var post struct {
				ID       int
				UserID   int
				Title    string
				Content  string
				Category string
			}
			err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Category)
			if err != nil {
				http.Error(w, "Failed to scan post", http.StatusInternalServerError)
				return
			}
			posts = append(posts, post)
		}

		// Parse the template
		tmpl, err := template.ParseFiles("./html/posts.html")
		if err != nil {
			http.Error(w, "Failed to parse template", http.StatusInternalServerError)
			return
		}

		// Execute the template with the posts data
		err = tmpl.Execute(w, posts)
		if err != nil {
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
