package main

import (
	"database/sql"
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
			ID       int
			UserID   int
			Content  string
			Likes    int
			Dislikes int
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
				ID       int
				UserID   int
				Content  string
				Likes    int
				Dislikes int
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
				ID       int
				UserID   int
				Content  string
				Likes    int
				Dislikes int
			}
			err := commentRows.Scan(&comment.ID, &comment.UserID, &comment.Content)
			if err != nil {
				http.Error(w, "Failed to scan comment: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Fetch likes and dislikes for comments
			err = db.QueryRow("SELECT COUNT(*) FROM reactions WHERE comment_id = ? AND type = 'like'", comment.ID).Scan(&comment.Likes)
			if err != nil {
				http.Error(w, "Failed to fetch comment likes: "+err.Error(), http.StatusInternalServerError)
				return
			}
			err = db.QueryRow("SELECT COUNT(*) FROM reactions WHERE comment_id = ? AND type = 'dislike'", comment.ID).Scan(&comment.Dislikes)
			if err != nil {
				http.Error(w, "Failed to fetch comment dislikes: "+err.Error(), http.StatusInternalServerError)
				return
			}

			post.Comments = append(post.Comments, comment)
		}

		// Add the post (with likes, dislikes, and comments) to the posts slice
		posts = append(posts, post)
	}

	// Log the number of posts and check for issues
	log.Printf("Fetched %d posts", len(posts))
	if len(posts) == 0 {
		log.Println("No posts found in the database.")
	}

	// Pass UserID to the template if logged in
	sessionCookie, err := r.Cookie("session_id")
	userID := 0
	if err == nil {
		// Retrieve the user ID from the session if available
		userID = sessions[sessionCookie.Value]
	}

	// Parse the template
	tmpl, err := template.ParseFiles("./html/posts.html")
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	// Execute the template with posts data and UserID
	err = tmpl.Execute(w, struct {
		Posts []struct {
			ID       int
			UserID   int
			Title    string
			Content  string
			Category string
			Likes    int
			Dislikes int
			Comments []struct {
				ID       int
				UserID   int
				Content  string
				Likes    int
				Dislikes int
			}
		}
		UserID int
	}{posts, userID})
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

		// Get the post_id or comment_id from the form
		postID := r.FormValue("post_id")
		commentID := r.FormValue("comment_id")
		reactionType := r.FormValue("type")

		var currentReact string
		var targetID string
		var isComment bool

		// Determine if it's a post or a comment
		if postID != "" {
			targetID = postID
			isComment = false
		} else if commentID != "" {
			targetID = commentID
			isComment = true
		} else {
			http.Error(w, "Invalid ID provided", http.StatusBadRequest)
			return
		}

		// Define the appropriate table and ID column based on whether it's a post or a comment
		var tableName, idColumn string
		if isComment {
			tableName = "reactions"
			idColumn = "comment_id"
		} else {
			tableName = "reactions"
			idColumn = "post_id"
		}

		// Check if the user has already reacted to the post/comment
		var res *sql.Rows
		if isComment {
			res, _ = db.Query(`SELECT type FROM reactions WHERE user_id = ? AND comment_id = ?;`, userID, targetID)
		} else {
			res, _ = db.Query(`SELECT type FROM reactions WHERE user_id = ? AND post_id = ?;`, userID, targetID)
		}

		if !res.Next() {
			// No previous reaction found, so insert the new reaction
			res.Close()
			_, err := db.Exec(fmt.Sprintf("INSERT INTO %s (%s, user_id, type) VALUES (?, ?, ?)", tableName, idColumn), targetID, userID, reactionType)
			if err != nil {
				http.Error(w, "Failed to react", http.StatusInternalServerError)
				return
			}
		} else {
			// A previous reaction exists, so toggle it if needed
			err := res.Scan(&currentReact)
			res.Close()
			if err != nil {
				http.Error(w, "Failed to fetch reaction", http.StatusInternalServerError)
				return
			}
			if (reactionType == "like" && currentReact == "dislike") || (reactionType == "dislike" && currentReact == "like") {
				// Remove the existing reaction
				if isComment {
					_, err = db.Exec(`DELETE FROM reactions WHERE user_id = ? AND comment_id = ?;`, userID, targetID)
				} else {
					_, err = db.Exec(`DELETE FROM reactions WHERE user_id = ? AND post_id = ?;`, userID, targetID)
				}
				if err != nil {
					http.Error(w, "Failed to remove existing reaction", http.StatusInternalServerError)
					return
				}
				// Insert the new reaction
				_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (%s, user_id, type) VALUES (?, ?, ?)", tableName, idColumn), targetID, userID, reactionType)
				if err != nil {
					http.Error(w, "Failed to insert new reaction", http.StatusInternalServerError)
					return
				}
			}
		}

		// Redirect back to the post or comment page
		if isComment {
			http.Redirect(w, r, "/posts#"+postID+"#comment-"+commentID, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/posts#"+postID, http.StatusSeeOther)
		}
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
			Likes    int
			Dislikes int
			Comments []struct {
				ID       int
				UserID   int
				Content  string
				Likes    int
				Dislikes int
			}
		}

		// Iterate over each post
		for rows.Next() {
			var post struct {
				ID       int
				UserID   int
				Title    string
				Content  string
				Category string
				Likes    int
				Dislikes int
				Comments []struct {
					ID       int
					UserID   int
					Content  string
					Likes    int
					Dislikes int
				}
			}

			// Scan the post data
			err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Category)
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
					ID       int
					UserID   int
					Content  string
					Likes    int
					Dislikes int
				}
				err := commentRows.Scan(&comment.ID, &comment.UserID, &comment.Content)
				if err != nil {
					http.Error(w, "Failed to scan comment: "+err.Error(), http.StatusInternalServerError)
					return
				}

				// Fetch likes and dislikes for comments
				err = db.QueryRow("SELECT COUNT(*) FROM reactions WHERE comment_id = ? AND type = 'like'", comment.ID).Scan(&comment.Likes)
				if err != nil {
					http.Error(w, "Failed to fetch comment likes: "+err.Error(), http.StatusInternalServerError)
					return
				}
				err = db.QueryRow("SELECT COUNT(*) FROM reactions WHERE comment_id = ? AND type = 'dislike'", comment.ID).Scan(&comment.Dislikes)
				if err != nil {
					http.Error(w, "Failed to fetch comment dislikes: "+err.Error(), http.StatusInternalServerError)
					return
				}

				post.Comments = append(post.Comments, comment)
			}

			// Add the post (with likes, dislikes, and comments) to the posts slice
			posts = append(posts, post)
		}

		// Log the number of posts and check for issues
		log.Printf("Fetched %d posts", len(posts))
		if len(posts) == 0 {
			log.Println("No posts found in the database.")
		}

		// Pass UserID to the template if logged in
		sessionCookie, err := r.Cookie("session_id")
		userID := 0
		if err == nil {
			// Retrieve the user ID from the session if available
			userID = sessions[sessionCookie.Value]
		}

		// Parse the template
		tmpl, err := template.ParseFiles("./html/posts.html")
		if err != nil {
			http.Error(w, "Failed to parse template", http.StatusInternalServerError)
			return
		}

		// Execute the template with posts data and UserID
		err = tmpl.Execute(w, struct {
			Posts []struct {
				ID       int
				UserID   int
				Title    string
				Content  string
				Category string
				Likes    int
				Dislikes int
				Comments []struct {
					ID       int
					UserID   int
					Content  string
					Likes    int
					Dislikes int
				}
			}
			UserID int
		}{posts, userID})
		if err != nil {
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			return
		}
	}
}
