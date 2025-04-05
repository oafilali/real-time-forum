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
	
	// Initialize the WebSocket hub
	handler.InitWebSocketHub()
	
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
	http.HandleFunc("/user/all", handler.GetAllUsersHandler)
	
	// WebSocket endpoint
	http.HandleFunc("/ws", handler.WebSocketHandler)
	
	// Add a handler for the root path - with improved content-type detection
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if this is an API request by examining headers, XHR flag, or query param
		isAPIRequest := strings.Contains(r.Header.Get("Accept"), "application/json") || 
						r.Header.Get("X-Requested-With") == "XMLHttpRequest" ||
						r.URL.Query().Get("api") == "true"
		
		log.Printf("Root path request: %s, isAPIRequest: %v, Accept: %s, X-Requested-With: %s, api param: %s", 
			r.URL.Path, isAPIRequest, r.Header.Get("Accept"), r.Header.Get("X-Requested-With"), r.URL.Query().Get("api"))
		
		if r.URL.Path == "/" && isAPIRequest {
			// If JSON is requested, use the HomeHandler to return JSON data
			log.Println("Calling HomeHandler for API request")
			handler.HomeHandler(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/static/") {
			// Let the static file server handle this
			return
		} else {
			// For all other routes when HTML is expected, serve the SPA index
			log.Printf("Serving SPA index.html for path: %s", r.URL.Path)
			http.ServeFile(w, r, "./web/templates/index.html")
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