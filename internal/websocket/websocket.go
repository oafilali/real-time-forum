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
	Send     chan []byte // Channel for outgoing messages
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
	SenderID   int    `json:"sender_id,omitempty"`
	ReceiverID int    `json:"receiverID"` // Match exactly with client-side JSON field name
	Content    string `json:"content,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
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
	log.Println("Starting WebSocket hub")
	for {
		select {
		case client := <-h.Register:
			h.mutex.Lock()
			log.Printf("Registering client: UserID=%d, Username=%s", client.UserID, client.Username)
			h.Clients[client.UserID] = client
			h.mutex.Unlock()
			h.broadcastUserList()
		case client := <-h.Unregister:
			h.mutex.Lock()
			if _, ok := h.Clients[client.UserID]; ok {
				log.Printf("Unregistering client: UserID=%d, Username=%s", client.UserID, client.Username)
				delete(h.Clients, client.UserID)
				close(client.Send)
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
	for userID, client := range h.Clients {
		users = append(users, map[string]interface{}{
			"id":       userID,
			"username": client.Username,
		})
	}

	message := map[string]interface{}{
		"type":  "user_list",
		"users": users,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling user list: %v", err)
		return
	}

	log.Printf("Broadcasting user list with %d users", len(users))
	for _, client := range h.Clients {
		select {
		case client.Send <- data:
			// Message sent successfully
		default:
			// Skip clients with full message queues
			log.Printf("Skipping client %d due to full message queue", client.UserID)
		}
	}
}

// ServeWs handles WebSocket requests from clients
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID int, username string) {
	log.Printf("Upgrading connection for user %d (%s)", userID, username)
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
		Send:     make(chan []byte, 256),
	}

	// Register client
	client.Hub.Register <- client
	log.Printf("Client registered: UserID=%d, Username=%s", client.UserID, client.Username)

	// Start goroutines for reading and writing
	go client.readPump()
	go client.writePump()
}

// readPump handles incoming messages from the client
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
		log.Printf("Client disconnected: UserID=%d", c.UserID)
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		log.Printf("Received message from user %d: %s", c.UserID, string(data))

		// Parse the raw JSON to get direct access to fields
		var jsonMap map[string]interface{}
		if err := json.Unmarshal(data, &jsonMap); err != nil {
			log.Printf("Error parsing message JSON: %v", err)
			continue
		}

		// Create a new message with explicit field access
		var message Message
		message.Type = jsonMap["type"].(string)
		
		// Explicitly extract receiverID as int
		if receiverIDFloat, ok := jsonMap["receiverID"].(float64); ok {
			message.ReceiverID = int(receiverIDFloat)
			log.Printf("Successfully extracted receiverID: %d", message.ReceiverID)
		} else {
			log.Printf("Warning: Could not extract receiverID from message")
			continue
		}
		
		// Extract content if present
		if content, ok := jsonMap["content"].(string); ok {
			message.Content = content
		}

		// Handle based on message type
		switch message.Type {
		case "message":
			handleChatMessage(c, message)
		case "get_history":
			handleHistoryRequest(c, message)
		}
	}
}

// handleChatMessage processes a chat message from a client
func handleChatMessage(c *Client, message Message) {
	// Explicitly set and log the key values to ensure they're correct
	senderID := c.UserID
	receiverID := message.ReceiverID
	content := message.Content

	// Log the explicit values
	log.Printf("Processing message with explicit values: SenderID=%d, ReceiverID=%d, Content=%s", 
		senderID, receiverID, content)
	
	// Store message in database with current timestamp
	timestamp := time.Now().Format(time.RFC3339)
	
	// Use explicit parameters for DB query to ensure correct values
	_, err := database.Db.Exec(
		"INSERT INTO private_messages (sender_id, receiver_id, content, timestamp) VALUES (?, ?, ?, ?)",
		senderID, receiverID, content, timestamp,
	)
	if err != nil {
		log.Printf("Error storing message in database: %v", err)
		return
	}
	
	// Create a completely new response message
	responseMsg := Message{
		Type:       "message",
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		Timestamp:  timestamp,
	}
	
	// Marshal fresh message object to avoid any reference issues
	respData, err := json.Marshal(responseMsg)
	if err != nil {
		log.Printf("Error marshaling response message: %v", err)
		return
	}
	
	// Send to recipient
	c.Hub.mutex.Lock()
	recipient, recipientExists := c.Hub.Clients[receiverID]
	c.Hub.mutex.Unlock()
	
	log.Printf("Looking for recipient with ID: %d (explicit value)", receiverID)
	
	if recipientExists {
		log.Printf("Recipient %d found, sending message", receiverID)
		select {
		case recipient.Send <- respData:
			log.Printf("Message successfully sent to recipient %d", receiverID)
		default:
			log.Printf("Failed to send message to recipient %d: channel full", receiverID)
		}
	} else {
		log.Printf("Recipient %d is not online", receiverID)
	}
	
	// Send confirmation back to sender
	select {
	case c.Send <- respData:
		log.Printf("Message confirmation sent back to sender %d", senderID)
	default:
		log.Printf("Failed to send confirmation to sender %d: channel full", senderID)
	}
}

// handleHistoryRequest handles a request for message history
func handleHistoryRequest(c *Client, message Message) {
	// Explicitly extract the user ID being requested
	otherUserID := message.ReceiverID
	
	log.Printf("History requested between users %d and %d", c.UserID, otherUserID)
	
	// Query for messages between these two users using explicit values
	rows, err := database.Db.Query(`
		SELECT sender_id, receiver_id, content, timestamp
		FROM private_messages
		WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
		ORDER BY timestamp DESC
		LIMIT 10
	`, c.UserID, otherUserID, otherUserID, c.UserID)
	
	if err != nil {
		log.Printf("Error querying message history: %v", err)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var timestamp string
		err := rows.Scan(&msg.SenderID, &msg.ReceiverID, &msg.Content, &timestamp)
		if err != nil {
			log.Printf("Error scanning message row: %v", err)
			continue
		}
		msg.Type = "message"
		msg.Timestamp = timestamp
		messages = append(messages, msg)
	}

	log.Printf("Found %d messages in history between users %d and %d", 
		len(messages), c.UserID, otherUserID)

	// Send history back to client
	response := map[string]interface{}{
		"type":     "history",
		"messages": messages,
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling history response: %v", err)
		return
	}
	
	select {
	case c.Send <- data:
		log.Printf("Message history sent to user %d", c.UserID)
	default:
		log.Printf("Failed to send history to user %d: channel full", c.UserID)
	}
}

// writePump handles sending messages to the client
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}