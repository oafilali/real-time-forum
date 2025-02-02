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


func homeHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		postRows, err := db.Query("SELECT id, title FROM posts")
	if err != nil {
		http.Error(w, "Failed to fetch posts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer postRows.Close()

	var posts []struct {
		ID       int
		Title    string
	}

	// Iterate over each post
	for postRows.Next() {
		var post struct {
			ID       int
			Title    string
		}

		// Scan the post data
		err := postRows.Scan(&post.ID, &post.Title)
		if err != nil {
			http.Error(w, "Failed to scan post: "+err.Error(), http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	// Parse the template
	tmpl, err := template.ParseFiles("./html/home.html")
	if err != nil {
		http.Error(w, "Failed to parse template: home"+err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the template with the posts data
	err = tmpl.Execute(w, posts)
	if err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
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

		http.Redirect(w, r, "/home", http.StatusFound)
	} else {
		http.ServeFile(w, r, "./html/login.html")
	}
}

// postHandler handles creating a new post
func createPostHandler(w http.ResponseWriter, r *http.Request) {
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

		var id int

		err = db.QueryRow("SELECT LAST_INSERT_ID()").Scan(&id)
		if err != nil {
			http.Error(w, "Failed to fetch post ID: "+err.Error(), http.StatusInternalServerError)
			return
		}

	
		http.Redirect(w, r, fmt.Sprintf("/posts?id=%d", id), http.StatusFound)
	} else {
		fmt.Print("test")
		http.ServeFile(w, r, "./html/createPost.html")
	}
}
// postsHandler displays a single post
func postHandler(w http.ResponseWriter, r *http.Request) {
    // Get the post ID from the URL query parameter
    postID := r.URL.Query().Get("id")
    if postID == "" {
        http.Error(w, "Post ID is missing", http.StatusBadRequest)
        return
    }

    // Fetch the post from the database using QueryRow for a single post
    type Post struct {
        ID       int
        Username string
        UserID   int
        Title    string
        Content  string
        Category string
        Likes    int
        Dislikes int
        Comments []struct {
            ID      int
            Username string
            UserID  int
            Content string
			Likes    int
			Dislikes int
        }
    }

	// Define a struct for the viewData that includes Post and UserID
type ViewData struct {
    Post   Post
    SessionID int
}

var post Post

    err := db.QueryRow("SELECT id, user_id, title, content, category FROM posts WHERE id = ?", postID).Scan(
        &post.ID, &post.UserID, &post.Title, &post.Content, &post.Category)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Post not found", http.StatusNotFound)
        } else {
            http.Error(w, "Failed to fetch post: "+err.Error(), http.StatusInternalServerError)
        }
        return
    }
	username := ""
	err = db.QueryRow("SELECT username FROM users WHERE id= ?", post.UserID).Scan(&username)
    if err != nil {
        http.Error(w, "Failed to fetch username: "+err.Error(), http.StatusInternalServerError)
        return
    }
	post.Username = username


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
    // Fetch comments along with usernames
    commentRows, err := db.Query(`
        SELECT c.id, c.user_id, c.content, u.username
        FROM comments c
        JOIN users u ON u.id = c.user_id
        WHERE c.post_id = ?`, post.ID)
    if err != nil {
        http.Error(w, "Failed to fetch comments: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer commentRows.Close()

    // Iterate over comments for this post
    for commentRows.Next() {
        var comment struct {
            ID      int
            Username string
            UserID  int
            Content string
			Likes    int
			Dislikes int
        }
        err := commentRows.Scan(&comment.ID, &comment.UserID, &comment.Content, &comment.Username)
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

   //Pass UserID to the template if logged in
	sessionCookie, err := r.Cookie("session_id")
	sessionID := 0
	if err == nil {
		// Retrieve the user ID from the session if available
		sessionID = sessions[sessionCookie.Value]
	}

	//fmt.Println("sessionID:", sessionID)

  viewData := ViewData{
        Post:   post,
        SessionID: sessionID, // Add the user ID
    }

//	fmt.Println("viewData:", viewData)
    // Parse the template
    tmpl, err := template.ParseFiles("./html/post.html")
    if err != nil {
        http.Error(w, "Failed to parse template", http.StatusInternalServerError)
        return
    }

    // Execute the template, passing in the post data
    err = tmpl.Execute(w, viewData)
    if err != nil {
        http.Error(w, "Failed to render template", http.StatusInternalServerError)
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
		} else if postID == ""{
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
