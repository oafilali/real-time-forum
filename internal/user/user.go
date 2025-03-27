package user

import (
	"fmt"
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/util"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// EmailExists checks if an email already exists in the database
func EmailExists(email string) (bool, error) {
	var count int
	err := database.Db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UsernameExists checks if a username already exists in the database
func UsernameExists(username string) (bool, error) {
	var count int
	err := database.Db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CheckEmailExists is the legacy method, kept for compatibility
func CheckEmailExists(w http.ResponseWriter, r *http.Request, email string) bool {
	exists, err := EmailExists(email)
	if err != nil {
		// If there's an error querying the database, handle it here
		util.ExecuteJSON(w, model.MsgData{"Database error"}, http.StatusInternalServerError)
		return true
	}
	return exists
}

// CheckUsernameExists is the legacy method, kept for compatibility
func CheckUsernameExists(w http.ResponseWriter, r *http.Request, username string) bool {
	exists, err := UsernameExists(username)
	if err != nil {
		util.ExecuteJSON(w, model.MsgData{"Database error"}, http.StatusInternalServerError)
		return true
	}
	return exists
}

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

func SaveUser(username, email, hashedPassword string) error {
	_, err := database.Db.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, hashedPassword,
	)
	return err
}

func AuthenticateUser(email, password string) (int, error) {
	var userID int
	var storedHash string
	err := database.Db.QueryRow(
		"SELECT id, password FROM users WHERE email = ?",
		email,
	).Scan(&userID, &storedHash)
	if err != nil {
		return 0, err
	}
	if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) != nil {
		return 0, fmt.Errorf("invalid password")
	}
	return userID, nil
}