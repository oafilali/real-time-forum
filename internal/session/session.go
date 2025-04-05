package session

import (
	"database/sql"
	"fmt"
	"forum/internal/database"
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
)

// CreateSession creates a new session for a user
func CreateSession(w http.ResponseWriter, userID int) error {
	log.Printf("Creating session for user ID: %d", userID)
	
	// Delete any existing sessions for this user
	_, err := database.Db.Exec("DELETE FROM sessions WHERE id = ?", userID)
	if err != nil {
		log.Printf("Error deleting existing sessions: %v", err)
		return err
	}

	// Generate a new session ID
	sessionID, err := uuid.NewV4()
	if err != nil {
		log.Printf("Error generating UUID: %v", err)
		return err
	}

	// Calculate expiration time
	expires := time.Now().Add(24 * time.Hour) // Extended to 24 hours for better UX
	
	// Insert new session into the database
	_, err = database.Db.Exec(
		"INSERT INTO sessions (session_id, id, expires_at) VALUES (?, ?, ?)",
		sessionID.String(), userID, expires,
	)
	if err != nil {
		log.Printf("Error inserting session: %v", err)
		return err
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID.String(),
		Expires:  expires,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	
	log.Printf("Session created successfully for user ID: %d", userID)
	return nil
}

// DeleteSession removes a session from the database
func DeleteSession(sessionID string) error {
	log.Printf("Deleting session: %s", sessionID)
	_, err := database.Db.Exec("DELETE FROM sessions WHERE session_id = ?", sessionID)
	if err != nil {
		log.Printf("Error deleting session: %v", err)
	}
	return err
}

// GetUserIDFromSession retrieves the user ID associated with a session
func GetUserIDFromSession(r *http.Request) (int, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, fmt.Errorf("no session cookie found: %v", err)
	}

	var userID int
	var expiresAt time.Time
	
	err = database.Db.QueryRow(
		"SELECT id, expires_at FROM sessions WHERE session_id = ?", cookie.Value,
	).Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("session not found in database")
	} else if err != nil {
		log.Printf("Database error checking session: %v", err)
		return 0, err
	}

	// Check if session is expired
	if time.Now().After(expiresAt) {
		log.Printf("Session expired for user ID: %d", userID)
		DeleteSession(cookie.Value)
		return 0, fmt.Errorf("session expired")
	}

	// Extend session on activity
	go extendSession(cookie.Value)
	
	return userID, nil
}

// Extends the session expiration time
func extendSession(sessionID string) {
	// Extend by 24 hours from now
	newExpiry := time.Now().Add(24 * time.Hour)
	
	_, err := database.Db.Exec(
		"UPDATE sessions SET expires_at = ? WHERE session_id = ?",
		newExpiry, sessionID,
	)
	
	if err != nil {
		log.Printf("Error extending session: %v", err)
	}
}

// CleanupExpiredSessions removes all expired sessions from the database
func CleanupExpiredSessions() {
	log.Println("Cleaning up expired sessions...")
	result, err := database.Db.Exec("DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP")
	if err != nil {
		log.Printf("Failed to clean expired sessions: %v", err)
		return
	}
	
	count, _ := result.RowsAffected()
	log.Printf("Cleaned up %d expired sessions", count)
}

// IsAuthenticated is a helper function to quickly check if a request is authenticated
func IsAuthenticated(r *http.Request) bool {
	userID, err := GetUserIDFromSession(r)
	return err == nil && userID > 0
}