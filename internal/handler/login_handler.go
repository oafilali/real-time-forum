package handler

import (
	"forum/internal/session"
	"forum/internal/user"
	"forum/internal/util"
	"html/template"
	"log"
	"net/http"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Validate email format
		if !util.IsValidEmail(email) {
			http.Redirect(w, r, "/login?error=Invalid%20email%20format", http.StatusFound)
			return
		}

		// Authenticate user
		userID, err := user.AuthenticateUser(email, password)
		if err != nil {
			// Log the error (invalid credentials)
			log.Println("Invalid credentials:", err)

			// Pass error message through query parameter to login page
			http.Redirect(w, r, "/login?error=Invalid%20email%20or%20password", http.StatusFound)
			return
		}

		// Create session
		if err := session.CreateSession(w, userID); err != nil {
			log.Println("Session creation failed:", err)
			// Respond with an error message if session creation fails
			http.Error(w, "Session creation failed", http.StatusInternalServerError)
			return
		}

		// Redirect to home page after successful login
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		// Handle GET requests for the login page
		errorMessage := r.URL.Query().Get("error")
		data := struct {
			ErrorMessage string
		}{
			ErrorMessage: errorMessage,
		}

		// Render the login template with the error message (if any)
		tmpl := template.Must(template.ParseFiles("./web/templates/login.html"))
		tmpl.Execute(w, data)
	}
}
