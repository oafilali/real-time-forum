package handler

import (
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/post"
	"forum/internal/session"
	"forum/internal/util"
	"net/http"
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
