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
	log.Println("Server starting on :8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

func initializeDatabase() {
	database.InitDB()
	session.CleanupExpiredSessions()
	log.Println("Database initialization complete")
}

func serveStaticFiles() {
	// Serve static files (CSS)
	// Serve static files (CSS, JS, images)
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
}

func registerHandlers() {
	// Register HTTP handlers
	http.HandleFunc("/", handler.HomeHandler)
	http.HandleFunc("/register", handler.RegisterHandler)
	http.HandleFunc("/login", handler.LoginHandler)
	http.HandleFunc("/createPost", handler.CreatePostHandler)
	http.HandleFunc("/comment", handler.CommentHandler)
	http.HandleFunc("/like", handler.LikeHandler)
	http.HandleFunc("/filter", handler.FilterHandler)
	http.HandleFunc("/post", handler.ViewPostHandler) // New route to display posts
	http.HandleFunc("/logout", handler.LogoutHandler) // New route to handle logout
}
