package handler

import (
	"fmt"
	"forum/internal/database"
	"forum/internal/post"
	"forum/internal/session"
	"forum/internal/util"
	"html/template"
	"net/http"
	"strings"
)

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

		if title == "" || content == "" {
			if util.ErrorCheckHandlers(w, r, "Post cannot be empty", fmt.Errorf("post cannot be empty"), http.StatusInternalServerError) {
				return
			}
			return
		}

		categoriesList := []string{
			"General",
			"Local News & Events",
			"Viking line",
			"Travel",
			"Sailing",
			"Cuisine & food",
			"Politics"}

		for _, userCategory := range r.Form["categories"] {
			catValid := false
			for _, categoryItem := range categoriesList {
				if categoryItem == userCategory {
					catValid = true
					break
				}
			}
			if !catValid {
				if util.ErrorCheckHandlers(w, r, "Invalid category", fmt.Errorf("invalid category"), http.StatusInternalServerError) {
					return
				}
				return
			}
		}

		if len(categories) == 0 {
			categories = "General"
		}

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
