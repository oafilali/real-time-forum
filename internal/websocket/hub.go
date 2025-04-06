package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	Clients    map[int]*Client
	
	// Register requests from the clients
	Register   chan *Client
	
	// Unregister requests from clients
	Unregister chan *Client
	
	// Mutex for thread-safe operations on clients map
	mutex      sync.Mutex
}

// Client represents a connected websocket client
type Client struct {
	UserID   int
	Username string
	Send     chan []byte
	Hub      *Hub
	Conn     *Connection
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[int]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run starts the hub and handles client registration/unregistration
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
    
    // Get all online users with their last message data
    usersWithData := make(map[int]struct{
        UserID    int
        Username  string
        LastMessage *time.Time
    })
    
    // First populate with basic user info
    for userID, client := range h.Clients {
        usersWithData[userID] = struct{
            UserID    int
            Username  string
            LastMessage *time.Time
        }{
            UserID:   userID,
            Username: client.Username,
            LastMessage: nil,
        }
    }
    
    // Prepare all users data for sending
    var users []map[string]interface{}
    
    for userID, userData := range usersWithData {
        // For each user, get their last conversations
        lastMessages, err := GetLastNMessagesForUsers(userID, 20)
        
        userInfo := map[string]interface{}{
            "id":       userData.UserID,
            "username": userData.Username,
        }
        
        // If we have message history, add it
        if err == nil && len(lastMessages) > 0 {
            messagesData := make(map[string]interface{})
            for otherUserID, msg := range lastMessages {
                messagesData[fmt.Sprintf("%d", otherUserID)] = map[string]interface{}{
                    "timestamp": msg.Timestamp,
                    "content":   msg.Content,
                    "sender_id": msg.SenderID,
                }
            }
            userInfo["lastMessages"] = messagesData
        }
        
        users = append(users, userInfo)
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

    log.Printf("Broadcasting user list with %d users", len(h.Clients))
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

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID int, message []byte) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	client, exists := h.Clients[userID]
	if !exists {
		return false
	}
	
	select {
	case client.Send <- message:
		return true
	default:
		return false
	}
}

// GetOnlineUsers returns a list of all online user IDs
func (h *Hub) GetOnlineUsers() []int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	var users []int
	for userID := range h.Clients {
		users = append(users, userID)
	}
	
	return users
}

// IsUserOnline checks if a user is currently online
func (h *Hub) IsUserOnline(userID int) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	_, exists := h.Clients[userID]
	return exists
}