package handler

import (
	"forum/internal/model"
	"forum/internal/session"
	"forum/internal/util"
	"log"
	"net/http"
)

// LogoutHandler handles user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		util.ExecuteJSON(w, model.MsgData{"Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		util.ExecuteJSON(w, model.MsgData{"No active session"}, http.StatusUnauthorized)
		return
	}

	if err := session.DeleteSession(cookie.Value); err != nil {
		log.Println("Logout failed:", err)
		util.ExecuteJSON(w, model.MsgData{"Logout failed"}, http.StatusInternalServerError)
		return
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", MaxAge: -1})

	util.ExecuteJSON(w, model.MsgData{"Logout successful"}, http.StatusOK)
}