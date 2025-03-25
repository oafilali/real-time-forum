package handler

import (
	"forum/internal/comment"
	"forum/internal/model"
	"forum/internal/session"
	"forum/internal/util"
	"net/http"
)

var ExecuteJSON = util.ExecuteJSON;
type MsgData = model.MsgData;

// commentHandler handles adding a comment to a post

func CommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		ExecuteJSON(w, MsgData{"Invalid request method"}, http.StatusMethodNotAllowed)
		return
	}

	sessionID, err := session.GetUserIDFromSession(r)
	if util.ErrorCheckHandlers(w, r, "Invalid session", err, http.StatusUnauthorized) {
		return
	}

	postID := r.FormValue("post_id")
	if postID == "" {
		util.ExecuteJSON(w, model.MsgData{"Post ID is missing"}, http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		util.ExecuteJSON(w, model.MsgData{"Content is missing"}, http.StatusBadRequest)
		return
	}

	if err := comment.AddComment(sessionID, postID, content); util.ErrorCheckHandlers(w, r, "Failed to add the comment", err, http.StatusInternalServerError) {
		return
	}

	// Send success response
	util.ExecuteJSON(w, model.MsgData{"Comment added successfully"}, http.StatusOK)
}
