package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Initialize the database
	initDB()
	defer db.Close()

	// Serve static files (CSS)
	http.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("./html"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))
	fmt.Println("Serving images from ./images/")
/*
	// Serve the home page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./html/home.html")
	})

	*/
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "html/favicon.ico")
	})

	// Register HTTP handlers
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/createPost", createPostHandler)
	http.HandleFunc("/comment", commentHandler)
	http.HandleFunc("/like", likeHandler)
	http.HandleFunc("/filter", filterHandler)
	http.HandleFunc("/post", postHandler)   // New route to display posts
	http.HandleFunc("/logout", logoutHandler) // New route to handle logout

	// Start the server
	fmt.Println("Server started at :8088")
	log.Fatal(http.ListenAndServe(":8088", nil))
}
