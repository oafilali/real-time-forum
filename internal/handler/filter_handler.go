package handler

import (
	"forum/internal/model"
	"database/sql"
	"forum/internal/database"
	"forum/internal/session"
	"forum/internal/util"
	"net/http"
)

// filterHandler handles filtering posts by category
func FilterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		util.ExecuteJSON(w, model.MsgData{"Invalid request method"}, http.StatusMethodNotAllowed)
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
		rows, err = database.Db.Query("SELECT id, title, category FROM posts WHERE user_id = ?", userID)
	} else if liked && userID > 0 {
		rows, err = database.Db.Query(`
            SELECT p.id, p.title, p.category 
            FROM posts p
            JOIN reactions r ON p.id = r.post_id 
            WHERE r.user_id = ? AND r.type = 'like'
        `, userID)
	} else if category != "" {
		rows, err = database.Db.Query("SELECT id, title, category FROM posts WHERE category LIKE ?", "%"+category+"%")
	} else {
		util.ExecuteJSON(w, model.MsgData{"Invalid filter request"}, http.StatusBadRequest)
		return
	}
	if util.ErrorCheckHandlers(w, r, "Failed to fetch posts", err, http.StatusInternalServerError) {
		return
	}
	defer rows.Close()

	// Collect filtered posts
	var posts []model.PostData // Assume you create a `PostData` struct in `model`
	for rows.Next() {
		var post model.PostData
		err = rows.Scan(&post.ID, &post.Title, &post.Category)
		if util.ErrorCheckHandlers(w, r, "Failed to scan post", err, http.StatusInternalServerError) {
			return
		}
		posts = append(posts, post)
	}

	// Send JSON response
	util.ExecuteJSON(w, struct {
		Category  string
		Posts     []model.PostData
		SessionID int
		Username  string
	}{
		Category:  category,
		Posts:     posts,
		SessionID: userID,
		Username:  username,
	}, http.StatusOK)
}
