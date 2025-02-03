package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type Post struct {
	ID       int
	Username string
	UserID   int
	Title    string
	Content  string
	Category string
	Likes    int
	Dislikes int
	Comments []comment
}

type comment struct {
	ID      int
	Username string
	UserID  int
	Content string
	Likes    int
	Dislikes int
}

type ViewData struct {
    Post   Post
    SessionID int
}

type posts struct {
	ID    int
	Title string
}

// sessions stores active user sessions
var sessions = make(map[string]int) // session ID -> user ID

func homeHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		allPosts, err := fetchPosts()
		if errorCheckHandlers(w, "Failed to load posts", err, http.StatusInternalServerError) {
			return
		}

		// Parse the template
		tmpl, err := template.ParseFiles("./html/home.html")
		if errorCheckHandlers(w, "Failed to parse the template", err, http.StatusInternalServerError) {
			return
		}

		// Execute the template with the posts data
		err = tmpl.Execute(w, allPosts)
		if errorCheckHandlers(w, "Failed to execute the template", err, http.StatusInternalServerError) {
			return
		}
	}
}

// registerHandler handles user registration
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		if checkEmailExists(w, email) {
			return
		}

		hashed, err := hashPassword(password)
		if errorCheckHandlers(w, "Password hashing failed", err, http.StatusInternalServerError) {
			return
		}

		if err := saveUser(username, email, hashed); errorCheckHandlers(w, "User registration failed", err, http.StatusInternalServerError) {
			return
		}

		http.Redirect(w, r, "/login", http.StatusFound)
	} else {
		http.ServeFile(w, r, "./html/register.html")
	}
}

// loginHandler handles user login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		email := r.FormValue("email")
		password := r.FormValue("password")

		userID, err := authenticateUser(email, password)
		if errorCheckHandlers(w, "Invalid credentials", err, http.StatusUnauthorized) {
			return
		}

		if err := createSession(w, userID); errorCheckHandlers(w, "Session creation failed", err, http.StatusInternalServerError) {
			return
		}

		http.Redirect(w, r, "/home", http.StatusFound)
	} else {
		http.ServeFile(w, r, "./html/login.html")
	}
}

// postHandler handles creating a new post
func createPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		userID, err := getUserIDFromSession(r)
		if errorCheckHandlers(w, "Invalid session", err, http.StatusUnauthorized) {
			return
		}

		title := r.FormValue("title")
		content := r.FormValue("content")
		category := r.FormValue("category")

		// Insert the post into the database
		if err := createPost(userID, title, content, category); errorCheckHandlers(w, "Post creation failed", err, http.StatusInternalServerError) {
			return
		}

		id, err := getPostId()
		if errorCheckHandlers(w, "Database issue", err, http.StatusInternalServerError) {
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/posts?id=%d", id), http.StatusFound)
	} else {
		fmt.Print("test")
		http.ServeFile(w, r, "./html/createPost.html")
	}
}

// postsHandler displays a single post
func viewPostHandler(w http.ResponseWriter, r *http.Request) {
    // Get the post ID from the URL query parameter
    postID := r.URL.Query().Get("id")
    if postID == "" {
        http.Error(w, "Post ID is missing", http.StatusBadRequest)
        return
    }

    post, err := fetchPost(postID)
    if errorCheckHandlers(w, "Failed to load the post", err, http.StatusInternalServerError) {
        return
    }

    post.Likes, post.Dislikes, err = fetchReactionsNumber(post.ID, false)
    if errorCheckHandlers(w, "Failed to load the reactions number", err, http.StatusInternalServerError) {
        return
    }

    // Fetch comments for this post
    post.Comments, err = fetchCommentsForPost(post.ID)
    if errorCheckHandlers(w, "Failed to load the comments", err, http.StatusInternalServerError) {
        return
    }

    // Pass UserID to the template if logged in
    sessionID, err := getUserIDFromSession(r)
    if err != nil {
        sessionID = 0 // If there's an error, set sessionID to 0
    }

    viewData := ViewData{
        Post:      post,
        SessionID: sessionID, // Add the user ID
    }

    // Parse the template
    tmpl, err := template.ParseFiles("./html/post.html")
    if errorCheckHandlers(w, "Failed to parse the template", err, http.StatusInternalServerError) {
        return
    }

    // Execute the template, passing in the post data
    err = tmpl.Execute(w, viewData)
    if errorCheckHandlers(w, "Failed to render the template", err, http.StatusInternalServerError) {
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
		//	log.Printf("Adding comment: post_id=%s, user_id=%d, content=%s\n", postID, userID, content)

		// Insert the comment into the database
		_, err = db.Exec("INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)", postID, userID, content)
		if err != nil {
			log.Println("Failed to add comment:", err)
			http.Error(w, "Failed to add comment", http.StatusInternalServerError)
			return
		}

		// Redirect back to the posts page
		http.Redirect(w, r, fmt.Sprintf("/post?id=%s", postID), http.StatusSeeOther)
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
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		postID := r.FormValue("post_id")
		commentID := r.FormValue("comment_id")
		reactionType := r.FormValue("type")

		var currentReact string
		var targetID string
		var isComment bool

		// Determine if it's a post or a comment

		if commentID != "" {
			targetID = commentID
			isComment = true
		} else if postID == "" {
			http.Error(w, "No ID provided", http.StatusBadRequest)
			return
		} else {
			targetID = postID
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

		//fmt.Println("postID ", postID)

		// Check if postID was found
		if postID == "" {
			http.Error(w, "Post ID is missing", http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/post?id=%s", postID), http.StatusSeeOther)
		/*	// Redirect back to the post or comment page
			if isComment {
				http.Redirect(w, r, "/posts#"+postID+"#comment-"+commentID, http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/posts#"+postID, http.StatusSeeOther)
			} */
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

// logoutHandler handles user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if errorCheckHandlers(w, "No active session", err, http.StatusUnauthorized) {
		return
	}

	if err := deleteSession(cookie.Value); errorCheckHandlers(w, "Logout failed", err, http.StatusInternalServerError) {
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", MaxAge: -1})
	fmt.Fprintf(w, "Logout successful")
	http.Redirect(w, r, "/home", http.StatusFound)
}

// filterHandler handles filtering posts by category
func filterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Get the category from the query parameter
		category := r.URL.Query().Get("category")
		if category == "" {
			http.Error(w, "Category parameter is missing", http.StatusBadRequest)
			return
		}

		// Query the posts for the specified category
		rows, err := db.Query("SELECT id, user_id, title, content, category FROM posts WHERE category = ?", category)
		if err != nil {
			http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Define a slice to hold the posts
		var posts []struct {
			ID       int
			UserID   int
			Title    string
			Content  string
			Category string
		}

		// Scan the rows into the posts slice
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

		// Prepare the data to pass to the template
		data := struct {
			Category string
			Posts    []struct {
				ID       int
				UserID   int
				Title    string
				Content  string
				Category string
			}
		}{
			Category: category,
			Posts:    posts,
		}

		// Parse and execute the template
		tmpl, err := template.ParseFiles("./html/category.html")
		if err != nil {
			http.Error(w, "Failed to parse template", http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
