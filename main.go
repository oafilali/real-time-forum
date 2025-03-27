package main

import (
	"forum/internal/database"
	"forum/internal/handler"
	"forum/internal/session"
	"log"
	"net/http"
	"strings"
)

func main() {
	// Initialize the database
	initializeDatabase()
	defer database.Db.Close()
	
	// Set up routes and start server
	startServer()
}

func startServer() {
	// Serve static files
	serveStaticFiles()
	
	log.Println("Server starting on :8080")
	
	// Register handlers using the original routes for data operations
	http.HandleFunc("/register", handler.RegisterHandler)
	http.HandleFunc("/login", handler.LoginHandler)
	http.HandleFunc("/logout", handler.LogoutHandler)
	http.HandleFunc("/createPost", handler.CreatePostHandler)
	http.HandleFunc("/comment", handler.CommentHandler)
	http.HandleFunc("/like", handler.LikeHandler)
	http.HandleFunc("/filter", handler.FilterHandler)
	http.HandleFunc("/post", handler.ViewPostHandler)
	http.HandleFunc("/user/status", handler.UserStatusHandler) // New endpoint for checking user status
	
	// Add a handler for the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check content type or Accept header for HTML vs JSON request
		acceptHeader := r.Header.Get("Accept")
		
		if r.URL.Path == "/" && (strings.Contains(acceptHeader, "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest") {
			// If JSON is requested, use the HomeHandler to return JSON data
			handler.HomeHandler(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/static/") {
			// Let the static file server handle this
			return
		} else {
			// For all other routes when HTML is expected, serve the SPA index
			http.ServeFile(w, r, "./web/static/index.html")
		}
	})
	
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initializeDatabase() {
	database.InitDB()
	session.CleanupExpiredSessions()
	log.Println("Database initialization complete")
}

func serveStaticFiles() {
	// Serve static files (CSS, JS, images)
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
}