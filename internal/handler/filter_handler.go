package handler

import (
	"database/sql"
	"forum/internal/database"
	"forum/internal/session"
	"forum/internal/util"
	"html/template"
	"net/http"
)

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
