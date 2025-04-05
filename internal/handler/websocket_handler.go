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
	log.Println("Initializing WebSocket Hub")
	WebSocketHub = websocket.NewHub()
	go WebSocketHub.Run()
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("WebSocket connection attempt")
	
	// Check for session
	userID, err := session.GetUserIDFromSession(r)
	if err != nil {
		log.Printf("WebSocket auth failed: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	log.Printf("WebSocket connection authenticated for user ID: %d", userID)

	// Get username
	var username string
	err = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		log.Printf("Error getting username for WebSocket: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	
	log.Printf("Serving WebSocket for user %d (%s)", userID, username)
	
	// Serve the WebSocket
	websocket.ServeWs(WebSocketHub, w, r, userID, username)
}