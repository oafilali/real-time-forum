package handler

import (
	"forum/internal/model"
	"forum/internal/user"
	"forum/internal/util"
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
	if user.CheckUsernameExists(w, r, username) {
		util.ExecuteJSON(w, model.MsgData{"Username already taken"}, http.StatusConflict)
		return
	}

	// Check if email already exists
	if user.CheckEmailExists(w, r, email) {
		if r.Header.Get("Accept") == "application/json" {
			util.ExecuteJSON(w, model.MsgData{"Email already taken"}, http.StatusConflict)
		} else {
			http.Redirect(w, r, "/register?error=Email%20already%20taken", http.StatusFound)
		}
		return
	}

	// Hash the password
	hashed, err := user.HashPassword(password)
	if util.ErrorCheckHandlers(w, r, "Password hashing failed", err, http.StatusInternalServerError) {
		return
	}

	// Save user to the database
	if err := user.SaveUser(username, email, hashed); util.ErrorCheckHandlers(w, r, "User registration failed", err, http.StatusInternalServerError) {
		return
	}

	// Return JSON or redirect to login page
	if r.Header.Get("Accept") == "application/json" {
		util.ExecuteJSON(w, model.MsgData{"Registration successful!"}, http.StatusOK)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}