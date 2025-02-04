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

func CreateSession(w http.ResponseWriter, userID int) error {
	_, err := database.Db.Exec("DELETE FROM sessions WHERE id = ?", userID)
	if err != nil {
		return err
	}

	sessionID, err := uuid.NewV4()
	if err != nil {
		return err
	}

	expires := time.Now().Add(2 * time.Hour)
	_, err = database.Db.Exec(
		"INSERT INTO sessions (session_id, id, expires_at) VALUES (?, ?, ?)",
		sessionID.String(), userID, expires,
	)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID.String(),
		Expires:  expires,
		HttpOnly: true,
	})
	return nil
}

func DeleteSession(sessionID string) error {
	_, err := database.Db.Exec("DELETE FROM sessions WHERE session_id = ?", sessionID)
	return err
}

func GetUserIDFromSession(r *http.Request) (int, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, fmt.Errorf("no session found")
	}

	var userID int
	var expiresAt time.Time
	err = database.Db.QueryRow(
		"SELECT id, expires_at FROM sessions WHERE session_id = ?", cookie.Value,
	).Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("session not found")
	} else if err != nil {
		return 0, err
	}

	if time.Now().After(expiresAt) {
		DeleteSession(cookie.Value)
		return 0, fmt.Errorf("session expired")
	}

	return userID, nil
}

func CleanupExpiredSessions() {
	_, err := database.Db.Exec("DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP")
	if err != nil {
		log.Println("Failed to clean expired sessions:", err)
	} else {
		log.Println("Expired sessions cleaned up successfully")
	}
}
