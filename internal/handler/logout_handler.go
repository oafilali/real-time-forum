package handler

import (
	"forum/internal/model"
	"forum/internal/session"
	"forum/internal/util"
	"net/http"
)

// logoutHandler handles user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if util.ErrorCheckHandlers(w, r, "No active session", err, http.StatusUnauthorized) {
		util.ExecuteJSON(w, model.MsgData{"No active session"}, http.StatusUnauthorized)
		return
	}

	if err := session.DeleteSession(cookie.Value); util.ErrorCheckHandlers(w, r, "Logout failed", err, http.StatusInternalServerError) {
		util.ExecuteJSON(w, model.MsgData{"Logout failed"}, http.StatusInternalServerError)
		return
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", MaxAge: -1})

	util.ExecuteJSON(w, model.MsgData{"Logout successful"}, http.StatusOK)
}
