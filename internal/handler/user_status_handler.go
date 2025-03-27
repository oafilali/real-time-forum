package handler

import (
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/session"
	"forum/internal/util"
	"log"
	"net/http"
)

// UserStatusHandler returns the current user's session information
func UserStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		util.ExecuteJSON(w, model.MsgData{"Invalid request method"}, http.StatusMethodNotAllowed)
		return
	}

	userID, err := session.GetUserIDFromSession(r)
	
	var username string
	if err == nil && userID > 0 {
		err = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
		if err != nil {
			log.Println("Error fetching username:", err)
			// Not returning error to client, just logging it
		}
	}
	
	data := struct {
		SessionID int    `json:"sessionID"`
		Username  string `json:"username"`
		LoggedIn  bool   `json:"loggedIn"`
	}{
		SessionID: userID,
		Username:  username,
		LoggedIn:  userID > 0,
	}
	
	util.ExecuteJSON(w, data, http.StatusOK)
}