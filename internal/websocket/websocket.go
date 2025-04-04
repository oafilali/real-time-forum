package websocket

import (
	"encoding/json"
	"forum/internal/database"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client represents a connected user
type Client struct {
	UserID   int
	Username string
	Conn     *websocket.Conn
	Hub      *Hub
}

// Hub manages all connected clients
type Hub struct {
	Clients    map[int]*Client
	Register   chan *Client
	Unregister chan *Client
	mutex      sync.Mutex
}

// Message represents a chat message
type Message struct {
	Type       string `json:"type"`
	SenderID   int    `json:"sender_id"`
	ReceiverID int    `json:"receiver_id"`
	Content    string `json:"content"`
	Timestamp  string `json:"timestamp"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[int]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run starts the WebSocket hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mutex.Lock()
			h.Clients[client.UserID] = client
			h.mutex.Unlock()
			h.broadcastUserList()
		case client := <-h.Unregister:
			h.mutex.Lock()
			if _, ok := h.Clients[client.UserID]; ok {
				delete(h.Clients, client.UserID)
				client.Conn.Close()
			}
			h.mutex.Unlock()
			h.broadcastUserList()
		}
	}
}

// broadcastUserList sends the list of online users to all clients
func (h *Hub) broadcastUserList() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var users []map[string]interface{}
	for _, client := range h.Clients {
		users = append(users, map[string]interface{}{
			"id":       client.UserID,
			"username": client.Username,
		})
	}

	message := map[string]interface{}{
		"type":  "user_list",
		"users": users,
	}

	data, _ := json.Marshal(message)
	for _, client := range h.Clients {
		err := client.Conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Error sending user list: %v", err)
		}
	}
}

// ServeWs handles WebSocket requests from clients
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID int, username string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}

	client := &Client{
		UserID:   userID,
		Username: username,
		Conn:     conn,
		Hub:      hub,
	}

	// Register client
	client.Hub.Register <- client

	// Start listening for messages
	go handleMessages(client)
}

// handleMessages processes incoming WebSocket messages
func handleMessages(client *Client) {
	defer func() {
		client.Hub.Unregister <- client
	}()

	for {
		_, data, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Handle based on message type
		switch message.Type {
		case "message":
			// Store message in database
			_, err := database.Db.Exec(
				"INSERT INTO private_messages (sender_id, receiver_id, content) VALUES (?, ?, ?)",
				client.UserID, message.ReceiverID, message.Content,
			)
			if err != nil {
				log.Printf("Error storing message: %v", err)
			}

			// Send to recipient if online
			message.SenderID = client.UserID
			message.Timestamp = time.Now().Format(time.RFC3339)
			
			respData, _ := json.Marshal(message)
			
			// Send to recipient if online
			client.Hub.mutex.Lock()
			if recipient, ok := client.Hub.Clients[message.ReceiverID]; ok {
				recipient.Conn.WriteMessage(websocket.TextMessage, respData)
			}
			client.Hub.mutex.Unlock()
			
			// Send back to sender as confirmation
			client.Conn.WriteMessage(websocket.TextMessage, respData)
			
		case "get_history":
			sendMessageHistory(client, message.ReceiverID)
		}
	}
}

// sendMessageHistory sends chat history to the client
func sendMessageHistory(client *Client, otherUserID int) {
	// Query for messages between these two users
	rows, err := database.Db.Query(`
		SELECT sender_id, receiver_id, content, timestamp
		FROM private_messages
		WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
		ORDER BY timestamp DESC
		LIMIT 10
	`, client.UserID, otherUserID, otherUserID, client.UserID)
	
	if err != nil {
		log.Printf("Error fetching message history: %v", err)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var timestamp string
		err := rows.Scan(&msg.SenderID, &msg.ReceiverID, &msg.Content, &timestamp)
		if err != nil {
			log.Printf("Error scanning message: %v", err)
			continue
		}
		msg.Type = "message"
		msg.Timestamp = timestamp
		messages = append(messages, msg)
	}

	// Send history back to client
	response := map[string]interface{}{
		"type":     "history",
		"messages": messages,
	}
	
	data, _ := json.Marshal(response)
	client.Conn.WriteMessage(websocket.TextMessage, data)
}