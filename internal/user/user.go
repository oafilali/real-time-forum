package user

import (
	"fmt"
	"forum/internal/database"
	"forum/internal/util"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func CheckEmailExists(w http.ResponseWriter, email string) bool {
	var count int
	err := database.Db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	if err != nil {
		// If there's an error querying the database, handle it here
		util.ErrorCheckHandlers(w, "Database error", err, http.StatusInternalServerError)
		return true
	}
	return count > 0
}

func CheckUsernameExists(w http.ResponseWriter, username string) bool {
	var count int
	err := database.Db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		util.ErrorCheckHandlers(w, "Database error", err, http.StatusInternalServerError)
		return true
	}
	return count > 0
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
