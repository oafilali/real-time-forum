package websocket

import (
	"encoding/json"
	"forum/internal/database"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections for now, can be more restrictive in production
	},
}

// Connection wraps a websocket connection
type Connection struct {
	ws *websocket.Conn
}

// ServeWs handles websocket requests from clients
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID int, username string) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	
	conn := &Connection{ws: ws}
	
	client := &Client{
		UserID:   userID,
		Username: username,
		Send:     make(chan []byte, 256),
		Hub:      hub,
		Conn:     conn,
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
		c.Conn.ws.Close()
		log.Printf("Client disconnected: UserID=%d", c.UserID)
	}()

	c.Conn.ws.SetReadLimit(4096)
	c.Conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.ws.SetPongHandler(func(string) error {
		c.Conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.Conn.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		log.Printf("Received message from user %d: %s", c.UserID, string(data))

		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			log.Printf("Error parsing message JSON: %v", err)
			continue
		}

		switch message.Type {
		case "message":
			handleChatMessage(c, message)
		case "get_history":
			handleHistoryRequest(c, message)
		}
	}
}

// handleChatMessage processes chat messages
func handleChatMessage(c *Client, message Message) {
	// Set sender ID and current timestamp
	senderID := c.UserID
	receiverID := message.ReceiverID
	content := message.Content
	
	// Store message in database
	if err := StoreMessage(senderID, receiverID, content); err != nil {
		log.Printf("Failed to store message: %v", err)
		return
	}
	
	// Create response message
	timestamp := time.Now().Format(time.RFC3339)
	responseMsg := Message{
		Type:       "message",
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		Timestamp:  timestamp,
		Username:   c.Username,
	}
	
	// Marshal response
	respData, err := json.Marshal(responseMsg)
	if err != nil {
		log.Printf("Error marshaling response message: %v", err)
		return
	}
	
	// Send to recipient if online
	sentToRecipient := c.Hub.SendToUser(receiverID, respData)
	log.Printf("Message to user %d delivered: %v", receiverID, sentToRecipient)
	
	// Send confirmation back to sender
	select {
	case c.Send <- respData:
		log.Printf("Message confirmation sent to sender %d", senderID)
	default:
		log.Printf("Failed to send confirmation to sender %d: channel full", senderID)
	}
}

// handleHistoryRequest handles requests for message history
func handleHistoryRequest(c *Client, message Message) {
	otherUserID := message.ReceiverID
	
	log.Printf("History requested between users %d and %d", c.UserID, otherUserID)
	
	// Get message history between these users
	messages, err := GetMessageHistory(c.UserID, otherUserID, 10)
	if err != nil {
		log.Printf("Error retrieving message history: %v", err)
		return
	}
	
	// Add usernames to messages
	for i := range messages {
		if messages[i].SenderID == c.UserID {
			messages[i].Username = c.Username
		} else {
			var username string
			err := database.Db.QueryRow("SELECT username FROM users WHERE id = ?", messages[i].SenderID).Scan(&username)
			if err != nil {
				messages[i].Username = "Unknown"
			} else {
				messages[i].Username = username
			}
		}
	}
	
	// Send history back to client
	response := CreateHistoryResponseMessage(messages)
	
	select {
	case c.Send <- response:
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
		c.Conn.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// The hub closed the channel
				c.Conn.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.ws.NextWriter(websocket.TextMessage)
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
			c.Conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}