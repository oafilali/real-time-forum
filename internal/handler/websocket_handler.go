package handler

import (
	"forum/internal/database"
	"forum/internal/session"
	"forum/internal/websocket"
	"log"
	"net/http"
)

var WebSocketHub *websocket.Hub

func InitWebSocketHub() {
	WebSocketHub = websocket.NewHub()
	go WebSocketHub.Run()
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := session.GetUserIDFromSession(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var username string
	err = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		log.Println("Error getting username:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	websocket.ServeWs(WebSocketHub, w, r, userID, username)
}