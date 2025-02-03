package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Initialize the database
	initializeDatabase()
	defer db.Close()
	startServer()
}

func startServer() {
	registerHandlers()
	serveStaticFiles()
	log.Println("Server starting on :8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

func initializeDatabase() {
	initDB()
	log.Println("Database initialization complete")
}

func serveStaticFiles() {
	// Serve static files (CSS)
	http.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("./html"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))
	fmt.Println("Serving images from ./images/")
}

func registerHandlers() {
	// Register HTTP handlers
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/createPost", createPostHandler)
	http.HandleFunc("/comment", commentHandler)
	http.HandleFunc("/like", likeHandler)
	http.HandleFunc("/filter", filterHandler)
	http.HandleFunc("/post", viewPostHandler) // New route to display posts
	http.HandleFunc("/logout", logoutHandler) // New route to handle logout
}
