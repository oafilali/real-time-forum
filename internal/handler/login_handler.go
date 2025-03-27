package handler

import (
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/session"
	"forum/internal/user"
	"forum/internal/util"
	"log"
	"net/http"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		util.ExecuteJSON(w, model.MsgData{"Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	
	email := r.FormValue("email")
	password := r.FormValue("password")

	if !util.IsValidEmail(email) {
		util.ExecuteJSON(w, model.MsgData{"Invalid email format"}, http.StatusBadRequest)
		return
	}

	userID, err := user.AuthenticateUser(email, password)
	if err != nil {
		log.Println("Invalid credentials:", err)
		util.ExecuteJSON(w, model.MsgData{"Invalid email or password"}, http.StatusUnauthorized)
		return
	}

	if err := session.CreateSession(w, userID); err != nil {
		log.Println("Session creation failed:", err)
		util.ExecuteJSON(w, model.MsgData{"Session creation failed"}, http.StatusInternalServerError)
		return
	}

	// Get username for response
	var username string
	_ = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)

	util.ExecuteJSON(w, struct {
		Message   string `json:"message"`
		SessionID int    `json:"sessionID"`
		Username  string `json:"username"`
	}{
		Message:   "Login successful",
		SessionID: userID,
		Username:  username,
	}, http.StatusOK)
}