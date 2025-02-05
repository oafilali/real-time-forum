package handler

import (
	"forum/internal/comment"
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/post"
	"forum/internal/reaction"
	"forum/internal/session"
	"forum/internal/util"
	"html/template"
	"net/http"
)

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
