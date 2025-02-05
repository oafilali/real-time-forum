package main

import (
	"forum/internal/database"
	"forum/internal/handler"
	"forum/internal/session"
	"forum/internal/util"
	"log"
	"net/http"
)

func main() {

	util.LoadTemplates() // Load all templates at startup

	// Initialize the database
	initializeDatabase()
	defer database.Db.Close()
	startServer()
}

func startServer() {
	registerHandlers()
	serveStaticFiles()
	util.LoadTemplates() // Load all templates at startup
	log.Println("Server starting on :8080")

	// Homepage handler
	http.HandleFunc("/", handler.HomeHandler)

	// Use custom 404 handler for undefined routes
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !isRegisteredRoute(r.URL.Path) && !isStaticFile(r.URL.Path) {
			handler.NotFoundHandler(w, r)
			return
		}
		http.DefaultServeMux.ServeHTTP(w, r)
	})))
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

func registerHandlers() {
	// Register HTTP handlers
	http.HandleFunc("/register", handler.RegisterHandler)
	http.HandleFunc("/login", handler.LoginHandler)
	http.HandleFunc("/createPost", handler.CreatePostHandler)
	http.HandleFunc("/comment", handler.CommentHandler)
	http.HandleFunc("/like", handler.LikeHandler)
	http.HandleFunc("/filter", handler.FilterHandler)
	http.HandleFunc("/post", handler.ViewPostHandler) // New route to display posts
	http.HandleFunc("/logout", handler.LogoutHandler) // New route to handle logout
}

func isRegisteredRoute(path string) bool {
	registeredRoutes := []string{"/", "/register", "/login", "/createPost", "/comment", "/like", "/filter", "/post", "/logout"}
	for _, route := range registeredRoutes {
		if path == route {
			return true
		}
	}
	return false
}

func isStaticFile(path string) bool {
	return len(path) >= 8 && path[:8] == "/static/"
}
