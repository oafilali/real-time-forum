package handler

import (
	"fmt"
	"forum/internal/comment"
	"forum/internal/session"
	"forum/internal/util"
	"net/http"
)

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
