package handler

import (
	"forum/internal/user"
	"forum/internal/util"
	"html/template"
	"net/http"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Check if username already exists
		if user.CheckUsernameExists(w, r, username) {
			http.Redirect(w, r, "/register?error=Username%20already%20taken", http.StatusFound)
			return
		}

		// Check if email already exists
		if user.CheckEmailExists(w, r, email) {
			http.Redirect(w, r, "/register?error=Email%20already%20taken", http.StatusFound)
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

		// Redirect to login page on successful registration
		http.Redirect(w, r, "/login", http.StatusFound)
	} else {
		// Handle GET request for the register page
		errorMessage := r.URL.Query().Get("error")
		data := struct {
			ErrorMessage string
		}{
			ErrorMessage: errorMessage,
		}

		// Render the register template with the error message (if any)
		tmpl := template.Must(template.ParseFiles("./web/templates/register.html"))
		tmpl.Execute(w, data)
	}
}
