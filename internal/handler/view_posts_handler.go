package handler

import (
	"forum/internal/comment"
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/post"
	"forum/internal/reaction"
	"forum/internal/session"
	"forum/internal/util"
	"net/http"
)

func ViewPostHandler(w http.ResponseWriter, r *http.Request) {
	// Get the post ID from the URL query parameter
	postID := r.URL.Query().Get("id")
	if postID == "" {
		ExecuteJSON(w, model.MsgData{"Missing PostID!"}, http.StatusBadRequest)
		return
	}

	post, err := post.FetchPost(postID)
	if err != nil {
		ExecuteJSON(w, MsgData{"Failed to load the post"}, http.StatusInternalServerError)
		return
	}

	post.Likes, post.Dislikes, err = reaction.FetchReactionsNumber(post.ID, false)
	if err != nil {
		ExecuteJSON(w, MsgData{"Failed to load the reactions number"}, http.StatusInternalServerError)
		return
	}

	// Fetch comments for this post
	post.Comments, err = comment.FetchCommentsForPost(post.ID)
	if err != nil {
		ExecuteJSON(w, MsgData{"Failed to load the comments"}, http.StatusInternalServerError)
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

	util.ExecuteJSON(w, postPageData, http.StatusOK)
}
