package handler

import (
	"database/sql"
	"fmt"
	"forum/internal/comment"
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/post"
	"forum/internal/reaction"
	"forum/internal/session"
	"forum/internal/user"
	"forum/internal/util"
	"html/template"
	"log"
	"net/http"
	"strings"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		userID, _ := session.GetUserIDFromSession(r)
		var username string
		if userID > 0 {
			_ = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
		}

		allPosts, err := post.FetchPosts()
		if util.ErrorCheckHandlers(w, r, "Failed to load posts", err, http.StatusInternalServerError) {
			return
		}

		data := model.Data{
			Posts:     allPosts,
			SessionID: userID,
			Username:  username,
		}

		err = util.Templates.ExecuteTemplate(w, "home.html", data)
		if util.ErrorCheckHandlers(w, r, "Failed to render the template", err, http.StatusInternalServerError) {
			return
		}

	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Check if username already exists
		if user.CheckUsernameExists(w, r, username) {
			http.Redirect(w, r, "/register?error=Username%20already%20taken", http.StatusFound)
			return
		}

		// Check if email already exists
		if user.CheckEmailExists(w, r, email) {
			http.Redirect(w, r, "/register?error=Email%20already%20taken", http.StatusFound)
			return
		}

		// Hash the password
		hashed, err := user.HashPassword(password)
		if util.ErrorCheckHandlers(w, r, "Password hashing failed", err, http.StatusInternalServerError) {
			return
		}

		// Save user to the database
		if err := user.SaveUser(username, email, hashed); util.ErrorCheckHandlers(w, r, "User registration failed", err, http.StatusInternalServerError) {
			return
		}

		// Redirect to login page on successful registration
		http.Redirect(w, r, "/login", http.StatusFound)
	} else {
		// Handle GET request for the register page
		errorMessage := r.URL.Query().Get("error")
		data := struct {
			ErrorMessage string
		}{
			ErrorMessage: errorMessage,
		}

		// Render the register template with the error message (if any)
		tmpl := template.Must(template.ParseFiles("./web/templates/register.html"))
		tmpl.Execute(w, data)
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Authenticate user
		userID, err := user.AuthenticateUser(email, password)
		if err != nil {
			// Log the error (invalid credentials)
			log.Println("Invalid credentials:", err)

			// Pass error message through query parameter to login page
			http.Redirect(w, r, "/login?error=Invalid%20email%20or%20password", http.StatusFound)
			return
		}

		// Create session
		if err := session.CreateSession(w, userID); err != nil {
			log.Println("Session creation failed:", err)
			// Respond with an error message if session creation fails
			http.Error(w, "Session creation failed", http.StatusInternalServerError)
			return
		}

		// Redirect to home page after successful login
		http.Redirect(w, r, "/home", http.StatusFound)
	} else {
		// Handle GET requests for the login page
		errorMessage := r.URL.Query().Get("error")
		data := struct {
			ErrorMessage string
		}{
			ErrorMessage: errorMessage,
		}

		// Render the login template with the error message (if any)
		tmpl := template.Must(template.ParseFiles("./web/templates/login.html"))
		tmpl.Execute(w, data)
	}
}

// postHandler handles creating a new post
func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		userID, err := session.GetUserIDFromSession(r)
		if util.ErrorCheckHandlers(w, r, "Invalid session", err, http.StatusUnauthorized) {
			return
		}

		title := r.FormValue("title")
		content := r.FormValue("content")
		categories := strings.Join(r.Form["categories"], ", ")

		// Insert the post into the database
		if err := post.CreatePost(userID, title, content, categories); util.ErrorCheckHandlers(w, r, "Post creation failed", err, http.StatusInternalServerError) {
			return
		}

		id, err := post.GetPostId()
		if util.ErrorCheckHandlers(w, r, "Database issue", err, http.StatusInternalServerError) {
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/post?id=%d", id), http.StatusFound)
	} else {
		sessionID, err := session.GetUserIDFromSession(r)
		if err != nil {
			sessionID = 0 // If there's an error, set sessionID to 0
		}

		var username string
		if sessionID > 0 {
			_ = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", sessionID).Scan(&username)
		}

		data := struct {
			SessionID int
			Username  string
		}{
			SessionID: sessionID,
			Username:  username,
		}

		tmpl, err := template.ParseFiles("./web/templates/createPost.html")
		if util.ErrorCheckHandlers(w, r, "Failed to parse the template", err, http.StatusInternalServerError) {
			return
		}

		err = tmpl.Execute(w, data)
		if util.ErrorCheckHandlers(w, r, "Failed to render the template", err, http.StatusInternalServerError) {
			return
		}
	}
}

// postsHandler displays a single post
func ViewPostHandler(w http.ResponseWriter, r *http.Request) {
	// Get the post ID from the URL query parameter
	postID := r.URL.Query().Get("id")
	if postID == "" {
		http.Error(w, "Post ID is missing", http.StatusBadRequest)
		return
	}

	post, err := post.FetchPost(postID)
	if util.ErrorCheckHandlers(w, r, "Failed to load the post", err, http.StatusInternalServerError) {
		return
	}

	post.Likes, post.Dislikes, err = reaction.FetchReactionsNumber(post.ID, false)
	if util.ErrorCheckHandlers(w, r, "Failed to load the reactions number", err, http.StatusInternalServerError) {
		return
	}

	// Fetch comments for this post
	post.Comments, err = comment.FetchCommentsForPost(post.ID)
	if util.ErrorCheckHandlers(w, r, "Failed to load the comments", err, http.StatusInternalServerError) {
		return
	}

	// Pass UserID to the template if logged in
	sessionID, err := session.GetUserIDFromSession(r)
	if err != nil {
		sessionID = 0 // If there's an error, set sessionID to 0
	}

	var username string
	if sessionID > 0 {
		_ = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", sessionID).Scan(&username)
	}

	postPageData := struct {
		Post      model.Post
		SessionID int
		Username  string
	}{
		Post:      post,
		SessionID: sessionID,
		Username:  username,
	}

	// Parse the template
	tmpl, err := template.ParseFiles("./web/templates/post.html")
	if util.ErrorCheckHandlers(w, r, "Failed to parse the template", err, http.StatusInternalServerError) {
		return
	}

	// Execute the template, passing in the post data
	err = tmpl.Execute(w, postPageData)
	if util.ErrorCheckHandlers(w, r, "Failed to render the template", err, http.StatusInternalServerError) {
		return
	}
}

// commentHandler handles adding a comment to a post
func CommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	sessionID, err := session.GetUserIDFromSession(r)
	if util.ErrorCheckHandlers(w, r, "Invalid session", err, http.StatusUnauthorized) {
		return
	}

	postID := r.FormValue("post_id")
	if postID == "" {
		http.Error(w, "Post ID is missing", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Content is missing", http.StatusBadRequest)
		return
	}

	if err := comment.AddComment(sessionID, postID, content); util.ErrorCheckHandlers(w, r, "Failed to add the comment", err, http.StatusInternalServerError) {
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/post?id=%s", postID), http.StatusFound)
}

// likeHandler handles liking or disliking a post or comment
func LikeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	sessionID, err := session.GetUserIDFromSession(r)

	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

	if util.ErrorCheckHandlers(w, r, "Invalid session", err, http.StatusUnauthorized) {
		return
	}

	itemID := r.FormValue("item_id")
	if itemID == "" {
		http.Error(w, "Item ID is missing", http.StatusBadRequest)
		return
	}

	isComment := r.FormValue("is_comment") == "true"
	reactionType := r.FormValue("type") // "like" or "dislike"

	if err := reaction.LikeItem(sessionID, itemID, isComment, reactionType); util.ErrorCheckHandlers(w, r, "Failed to like the item", err, http.StatusInternalServerError) {
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusFound)
}

// logoutHandler handles user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if util.ErrorCheckHandlers(w, r, "No active session", err, http.StatusUnauthorized) {
		return
	}

	if err := session.DeleteSession(cookie.Value); util.ErrorCheckHandlers(w, r, "Logout failed", err, http.StatusInternalServerError) {
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", MaxAge: -1})
	http.Redirect(w, r, "/home", http.StatusFound)
}

// filterHandler handles filtering posts by category
func FilterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := session.GetUserIDFromSession(r) // Get logged-in user ID (if any)

	var username string
	if userID > 0 {
		_ = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	}

	category := r.URL.Query().Get("category")
	userCreated := r.URL.Query().Get("user_created") == "true"
	liked := r.URL.Query().Get("liked") == "true"

	var rows *sql.Rows
	var err error

	if userCreated && userID > 0 {
		// Fetch posts created by the logged-in user
		rows, err = database.Db.Query("SELECT id, title, category FROM posts WHERE user_id = ?", userID)
	} else if liked && userID > 0 {
		// Fetch posts liked by the logged-in user
		rows, err = database.Db.Query(`
            SELECT p.id, p.title, p.category 
            FROM posts p
            JOIN reactions r ON p.id = r.post_id 
            WHERE r.user_id = ? AND r.type = 'like'
        `, userID)
	} else if category != "" {
		// Fetch posts by category
		rows, err = database.Db.Query("SELECT id, title, category FROM posts WHERE category LIKE ?", "%"+category+"%")
	} else {
		// Invalid filter request
		http.Error(w, "Invalid filter request", http.StatusBadRequest)
		return
	}
	if util.ErrorCheckHandlers(w, r, "Failed to fetch posts", err, http.StatusInternalServerError) {
		return
	}
	defer rows.Close()

	// Collect filtered posts
	var posts []struct {
		ID       int
		Title    string
		Category string
	}

	for rows.Next() {
		var post struct {
			ID       int
			Title    string
			Category string
		}
		err = rows.Scan(&post.ID, &post.Title, &post.Category)
		if util.ErrorCheckHandlers(w, r, "Failed to scan post", err, http.StatusInternalServerError) {
			return
		}
		posts = append(posts, post)
	}

	// Prepare data for the template
	data := struct {
		Category string
		Posts    []struct {
			ID       int
			Title    string
			Category string
		}
		SessionID int
		Username  string
	}{
		Category:  category,
		Posts:     posts,
		SessionID: userID,
		Username:  username,
	}

	// Render the template
	tmpl, err := template.ParseFiles("./web/templates/category.html")
	if util.ErrorCheckHandlers(w, r, "Failed to parse template", err, http.StatusInternalServerError) {
		return
	}

	err = tmpl.Execute(w, data)
	if util.ErrorCheckHandlers(w, r, "Failed to execute template", err, http.StatusInternalServerError) {
		return
	}
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	util.ErrorHandler(w, r, http.StatusNotFound, "Page Not Found")
}
