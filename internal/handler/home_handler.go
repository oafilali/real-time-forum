package handler

import (
	"fmt"
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/post"
	"forum/internal/session"
	"forum/internal/util"
	"log"
	"net/http"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		util.ExecuteJSON(w, model.MsgData{"Invalid request method"}, http.StatusMethodNotAllowed)
		return
	}

    sessionID, err := session.GetUserIDFromSession(r)
    fmt.Println(sessionID)
    if err != nil || sessionID == 0 {
        log.Println("Unauthorized access attempt to view post")
        util.ExecuteJSON(w, model.MsgData{"Unauthorized: Please log in to view posts"}, http.StatusUnauthorized)
        return
    }
	
	// Enhanced debugging with all headers that might be relevant
	log.Printf("HomeHandler called with headers - Accept: %s, X-Requested-With: %s, Query params: %v",
		r.Header.Get("Accept"), r.Header.Get("X-Requested-With"), r.URL.Query())
	
	userID, _ := session.GetUserIDFromSession(r)
	var username string
	if userID > 0 {
		_ = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	}

	allPosts, err := post.FetchPosts()
	if err != nil {
		log.Println("Failed to load posts:", err)
		util.ExecuteJSON(w, model.MsgData{"Failed to load posts"}, http.StatusInternalServerError)
		return
	}
	
	// Debug the fetched posts with more details
	log.Printf("Fetched %d posts for home page, returning JSON response", len(allPosts))
	
	// Debugging for the first post if available
	if len(allPosts) > 0 {
		log.Printf("First post sample: ID=%d, Title=%s", allPosts[0].ID, allPosts[0].Title)
	}
	
	if len(allPosts) > 5 {
		allPosts = allPosts[:5]
	}

	data := model.Data{
		Posts:     allPosts,
		SessionID: userID,
		Username:  username,
	}

	// Set proper content type header
	w.Header().Set("Content-Type", "application/json")
	
	// Return JSON response
	log.Println("Returning JSON response for home page")
	util.ExecuteJSON(w, data, http.StatusOK)
}