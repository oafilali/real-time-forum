package main

import (
	"database/sql"
	"fmt"
	"html/template"
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
	ID       int
	Username string
	UserID   int
	Content  string
	Likes    int
	Dislikes int
}

type postPageData struct {
	Post      Post
	SessionID int
}

type homePageData struct {
	ID       int
	Title    string
	Likes    int
	Dislikes int
}

type data struct {
	Posts     []homePageData
	SessionID int
	Username  string
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		userID, _ := getUserIDFromSession(r)
		var username string
		if userID > 0 {
			_ = db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
		}

		allPosts, err := fetchPosts()
		if errorCheckHandlers(w, "Failed to load posts", err, http.StatusInternalServerError) {
			return
		}

		data := data{
			Posts:     allPosts,
			SessionID: userID,
			Username:  username,
		}

		tmpl, err := template.ParseFiles("./html/home.html")
		if errorCheckHandlers(w, "Failed to parse the template", err, http.StatusInternalServerError) {
			return
		}

		err = tmpl.Execute(w, data)
		if errorCheckHandlers(w, "Failed to execute the template", err, http.StatusInternalServerError) {
			return
		}
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
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

        http.Redirect(w, r, fmt.Sprintf("/post?id=%d", id), http.StatusFound)
    } else {
        sessionID, err := getUserIDFromSession(r)
        if err != nil {
            sessionID = 0 // If there's an error, set sessionID to 0
        }

        var username string
        if sessionID > 0 {
            _ = db.QueryRow("SELECT username FROM users WHERE id = ?", sessionID).Scan(&username)
        }

        data := struct {
            SessionID int
            Username  string
        }{
            SessionID: sessionID,
            Username:  username,
        }

        tmpl, err := template.ParseFiles("./html/createPost.html")
        if errorCheckHandlers(w, "Failed to parse the template", err, http.StatusInternalServerError) {
            return
        }

        err = tmpl.Execute(w, data)
        if errorCheckHandlers(w, "Failed to render the template", err, http.StatusInternalServerError) {
            return
        }
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

    var username string
    if sessionID > 0 {
        _ = db.QueryRow("SELECT username FROM users WHERE id = ?", sessionID).Scan(&username)
    }

    postPageData := struct {
        Post      Post
        SessionID int
        Username  string
    }{
        Post:      post,
        SessionID: sessionID,
        Username:  username,
    }

    // Parse the template
    tmpl, err := template.ParseFiles("./html/post.html")
    if errorCheckHandlers(w, "Failed to parse the template", err, http.StatusInternalServerError) {
        return
    }

    // Execute the template, passing in the post data
    err = tmpl.Execute(w, postPageData)
    if errorCheckHandlers(w, "Failed to render the template", err, http.StatusInternalServerError) {
        return
    }
}

// commentHandler handles adding a comment to a post
func commentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	sessionID, err := getUserIDFromSession(r)
	if errorCheckHandlers(w, "Invalid session", err, http.StatusUnauthorized) {
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

	if err := addComment(sessionID, postID, content); errorCheckHandlers(w, "Failed to add the comment", err, http.StatusInternalServerError) {
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/post?id=%s", postID), http.StatusFound)
}

// likeHandler handles liking or disliking a post or comment
func likeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	sessionID, err := getUserIDFromSession(r)
	if errorCheckHandlers(w, "Invalid session", err, http.StatusUnauthorized) {
		return
	}

	itemID := r.FormValue("item_id")
	if itemID == "" {
		http.Error(w, "Item ID is missing", http.StatusBadRequest)
		return
	}

	isComment := r.FormValue("is_comment") == "true"
	reactionType := r.FormValue("type") // "like" or "dislike"

	if err := likeItem(sessionID, itemID, isComment, reactionType); errorCheckHandlers(w, "Failed to like the item", err, http.StatusInternalServerError) {
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusFound)
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
	http.Redirect(w, r, "/home", http.StatusFound)
}

// filterHandler handles filtering posts by category
func filterHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    userID, _ := getUserIDFromSession(r) // Get logged-in user ID (if any)

    var username string
    if userID > 0 {
        _ = db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
    }

    category := r.URL.Query().Get("category")
    userCreated := r.URL.Query().Get("user_created") == "true"
    liked := r.URL.Query().Get("liked") == "true"

    var rows *sql.Rows
    var err error

    if userCreated && userID > 0 {
        // Fetch posts created by the logged-in user
        rows, err = db.Query("SELECT id, title, category FROM posts WHERE user_id = ?", userID)
    } else if liked && userID > 0 {
        // Fetch posts liked by the logged-in user
        rows, err = db.Query(`
            SELECT p.id, p.title, p.category 
            FROM posts p
            JOIN reactions r ON p.id = r.post_id 
            WHERE r.user_id = ? AND r.type = 'like'
        `, userID)
    } else if category != "" {
        // Fetch posts by category
        rows, err = db.Query("SELECT id, title, category FROM posts WHERE category = ?", category)
    } else {
        // Invalid filter request
        http.Error(w, "Invalid filter request", http.StatusBadRequest)
        return
    }

    if err != nil {
        http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
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
        if err := rows.Scan(&post.ID, &post.Title, &post.Category); err != nil {
            http.Error(w, "Failed to scan post", http.StatusInternalServerError)
            return
        }
        posts = append(posts, post)
    }

    // Prepare data for the template
    data := struct {
        Category  string
        Posts     []struct {
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
    tmpl, err := template.ParseFiles("./html/category.html")
    if err != nil {
        http.Error(w, "Failed to parse template", http.StatusInternalServerError)
        return
    }

    if err := tmpl.Execute(w, data); err != nil {
        http.Error(w, "Failed to execute template", http.StatusInternalServerError)
    }
}
