package handler

import (
	"forum/internal/reaction"
	"forum/internal/session"
	"forum/internal/util"
	"net/http"
)

// likeHandler handles liking or disliking a post or comment
func LikeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	sessionID, err := session.GetUserIDFromSession(r)

	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

	if util.ErrorCheckHandlers(w, r, "Invalid session", err, http.StatusUnauthorized) {
		return
	}

	itemID := r.FormValue("item_id")
	if itemID == "" {
		http.Error(w, "Item ID is missing", http.StatusBadRequest)
		return
	}

	isComment := r.FormValue("is_comment") == "true"
	reactionType := r.FormValue("type") // "like" or "dislike"

	if err := reaction.LikeItem(sessionID, itemID, isComment, reactionType); util.ErrorCheckHandlers(w, r, "Failed to like the item", err, http.StatusInternalServerError) {
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusFound)
}
