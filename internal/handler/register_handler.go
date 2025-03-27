package handler

import (
	"forum/internal/model"
	"forum/internal/user"
	"forum/internal/util"
	"log"
	"net/http"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		util.ExecuteJSON(w, model.MsgData{"Invalid request method"}, http.StatusMethodNotAllowed)
		return
	}
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate email format
	if !util.IsValidEmail(email) {
		util.ExecuteJSON(w, model.MsgData{"Invalid email format"}, http.StatusBadRequest)
		return
	}

	// Check if username already exists
	exists, err := user.UsernameExists(username)
	if err != nil {
		log.Println("Database error checking username:", err)
		util.ExecuteJSON(w, model.MsgData{"Database error"}, http.StatusInternalServerError)
		return
	}
	if exists {
		util.ExecuteJSON(w, model.MsgData{"Username already taken"}, http.StatusConflict)
		return
	}

	// Check if email already exists
	exists, err = user.EmailExists(email)
	if err != nil {
		log.Println("Database error checking email:", err)
		util.ExecuteJSON(w, model.MsgData{"Database error"}, http.StatusInternalServerError)
		return
	}
	if exists {
		util.ExecuteJSON(w, model.MsgData{"Email already taken"}, http.StatusConflict)
		return
	}

	// Hash the password
	hashed, err := user.HashPassword(password)
	if err != nil {
		log.Println("Password hashing failed:", err)
		util.ExecuteJSON(w, model.MsgData{"Password hashing failed"}, http.StatusInternalServerError)
		return
	}

	// Save user to the database
	if err := user.SaveUser(username, email, hashed); err != nil {
		log.Println("User registration failed:", err)
		util.ExecuteJSON(w, model.MsgData{"User registration failed"}, http.StatusInternalServerError)
		return
	}

	// Return JSON response
	util.ExecuteJSON(w, model.MsgData{"Registration successful!"}, http.StatusOK)
}