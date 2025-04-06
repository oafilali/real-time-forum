package websocket

import (
	"encoding/json"
	"forum/internal/database"
	"log"
	"time"
)

// Message represents a chat message
type Message struct {
	Type       string `json:"type"`
	SenderID   int    `json:"sender_id,omitempty"`
	ReceiverID int    `json:"receiverID"` 
	Content    string `json:"content,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
	Username   string `json:"username,omitempty"`
}

// StoreMessage saves a message to the database
func StoreMessage(senderID, receiverID int, content string) error {
	timestamp := time.Now().Format(time.RFC3339)
	
	_, err := database.Db.Exec(
		"INSERT INTO private_messages (sender_id, receiver_id, content, timestamp) VALUES (?, ?, ?, ?)",
		senderID, receiverID, content, timestamp,
	)
	if err != nil {
		log.Printf("Error storing message in database: %v", err)
		return err
	}
	
	return nil
}

// GetMessageHistory retrieves message history between two users
func GetMessageHistory(userID1, userID2 int, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	
	rows, err := database.Db.Query(`
		SELECT sender_id, receiver_id, content, timestamp
		FROM private_messages
		WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
		ORDER BY timestamp DESC
		LIMIT ?
	`, userID1, userID2, userID2, userID1, limit)
	
	if err != nil {
		log.Printf("Error querying message history: %v", err)
		return nil, err
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
		
		// Get username for this message
		var username string
		err = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", msg.SenderID).Scan(&username)
		if err != nil {
			log.Printf("Error getting username for message: %v", err)
			username = "Unknown"
		}
		msg.Username = username
		
		messages = append(messages, msg)
	}

	// Reverse the order so newest messages are at the end
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMoreMessageHistory retrieves older message history before a specified timestamp
func GetMoreMessageHistory(userID1, userID2 int, before string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	
	// Parse the "before" timestamp
	var beforeTime time.Time
	var err error
	if before != "" {
		beforeTime, err = time.Parse(time.RFC3339, before)
		if err != nil {
			log.Printf("Error parsing timestamp: %v", err)
			// If timestamp is invalid, just use current time
			beforeTime = time.Now()
		}
	} else {
		// If no "before" timestamp, use current time
		beforeTime = time.Now()
	}
	
	// Query for older messages
	rows, err := database.Db.Query(`
		SELECT sender_id, receiver_id, content, timestamp
		FROM private_messages
		WHERE ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))
		AND timestamp < ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, userID1, userID2, userID2, userID1, beforeTime.Format(time.RFC3339), limit)
	
	if err != nil {
		log.Printf("Error querying more message history: %v", err)
		return nil, err
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
		
		// Get username for this message
		var username string
		err = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", msg.SenderID).Scan(&username)
		if err != nil {
			log.Printf("Error getting username for message: %v", err)
			username = "Unknown"
		}
		msg.Username = username
		
		messages = append(messages, msg)
	}

	// Reverse the order so newest messages are at the end
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// CreateUserListMessage creates a message with the current list of online users
func CreateUserListMessage(clients map[int]*Client) []byte {
	var users []map[string]interface{}
	
	for userID, client := range clients {
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
		return nil
	}
	
	return data
}

// CreateHistoryResponseMessage creates a message containing chat history
func CreateHistoryResponseMessage(messages []Message) []byte {
	response := map[string]interface{}{
		"type":     "history",
		"messages": messages,
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling history response: %v", err)
		return nil
	}
	
	return data
}

// CreateMoreHistoryResponseMessage creates a message containing additional chat history
func CreateMoreHistoryResponseMessage(messages []Message) []byte {
	response := map[string]interface{}{
		"type":     "more_history",
		"messages": messages,
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling more history response: %v", err)
		return nil
	}
	
	return data
}

// GetLastNMessagesForUsers retrieves the last N messages between all user pairs
// Useful for sorting the user list by most recent interaction
func GetLastNMessagesForUsers(userID int, n int) (map[int]Message, error) {
	results := make(map[int]Message)
	
	// Query for the most recent message with each user
	rows, err := database.Db.Query(`
		SELECT p1.*, u.username
		FROM (
			SELECT 
				pm.*, 
				CASE
					WHEN pm.sender_id = ? THEN pm.receiver_id
					WHEN pm.receiver_id = ? THEN pm.sender_id
				END as other_user_id,
				ROW_NUMBER() OVER (
					PARTITION BY 
						CASE
							WHEN pm.sender_id = ? THEN pm.receiver_id
							WHEN pm.receiver_id = ? THEN pm.sender_id
						END
					ORDER BY pm.timestamp DESC
				) as row_num
			FROM private_messages pm
			WHERE pm.sender_id = ? OR pm.receiver_id = ?
		) p1
		JOIN users u ON u.id = p1.sender_id
		WHERE p1.row_num = 1
		ORDER BY p1.timestamp DESC
		LIMIT ?
	`, userID, userID, userID, userID, userID, userID, n)
	
	if err != nil {
		log.Printf("Error querying last messages: %v", err)
		return results, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var msg Message
		var otherUserID int
		var rowNum int
		var timestamp string
		
		err := rows.Scan(
			&msg.SenderID, 
			&msg.ReceiverID, 
			&msg.Content, 
			&timestamp,
			&otherUserID,
			&rowNum,
			&msg.Username,
		)
		
		if err != nil {
			log.Printf("Error scanning message row: %v", err)
			continue
		}
		
		msg.Type = "message"
		msg.Timestamp = timestamp
		results[otherUserID] = msg
	}
	
	return results, nil
}

// FormatMessage sanitizes and formats a message for display
func FormatMessage(message *Message) *Message {
	// Add any message formatting/sanitization here
	
	// Set timestamp if not already set
	if message.Timestamp == "" {
		message.Timestamp = time.Now().Format(time.RFC3339)
	}
	
	return message
}