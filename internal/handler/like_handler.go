package handler

import (
	"forum/internal/model"
	"forum/internal/reaction"
	"forum/internal/session"
	"forum/internal/util"
	"net/http"
)

// likeHandler handles liking or disliking a post or comment
func LikeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		util.ExecuteJSON(w, model.MsgData{"Invalid request method"}, http.StatusMethodNotAllowed)
		return
	}

	sessionID, err := session.GetUserIDFromSession(r)
	if err != nil {
		util.ExecuteJSON(w, model.MsgData{"Invalid session, please log in"}, http.StatusUnauthorized)
		return
	}

	itemID := r.FormValue("item_id")
	if itemID == "" {
		util.ExecuteJSON(w, model.MsgData{"Item ID is missing"}, http.StatusBadRequest)
		return
	}

	isComment := r.FormValue("is_comment") == "true"
	reactionType := r.FormValue("type") // "like" or "dislike"

	if err := reaction.LikeItem(sessionID, itemID, isComment, reactionType); util.ErrorCheckHandlers(w, r, "Failed to like the item", err, http.StatusInternalServerError) {
		return
	}

	util.ExecuteJSON(w, model.MsgData{"Reaction recorded successfully"}, http.StatusOK)
}
