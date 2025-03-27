package handler

import (
	"forum/internal/comment"
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/post"
	"forum/internal/reaction"
	"forum/internal/session"
	"forum/internal/util"
	"log"
	"net/http"
)

func ViewPostHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        util.ExecuteJSON(w, model.MsgData{"Invalid request method"}, http.StatusMethodNotAllowed)
        return
    }

    // Get the post ID from the URL query parameter
    postID := r.URL.Query().Get("id")
    log.Printf("ViewPostHandler called with postID: %s", postID)
    
    if postID == "" || postID == "undefined" || postID == "null" {
        log.Println("Missing or invalid PostID in request:", postID)
        util.ExecuteJSON(w, model.MsgData{"Missing or invalid PostID"}, http.StatusBadRequest)
        return
    }
    
    log.Printf("Fetching post with ID: %s", postID)
    
    post, err := post.FetchPost(postID)
    if err != nil {
        log.Printf("Failed to load the post with ID %s: %v", postID, err)
        util.ExecuteJSON(w, model.MsgData{"Failed to load the post"}, http.StatusInternalServerError)
        return
    }

    post.Likes, post.Dislikes, err = reaction.FetchReactionsNumber(post.ID, false)
    if err != nil {
        log.Printf("Failed to load the reactions for post ID %d: %v", post.ID, err)
        // Continue anyway, just with zero likes/dislikes
        post.Likes = 0
        post.Dislikes = 0
    }

    // Fetch comments for this post
    post.Comments, err = comment.FetchCommentsForPost(post.ID)
    if err != nil {
        log.Printf("Failed to load comments for post ID %d: %v", post.ID, err)
        // Continue anyway, just with empty comments
        post.Comments = []model.Comment{}
    }

    // Pass UserID to the template if logged in
    sessionID, err := session.GetUserIDFromSession(r)
    if err != nil {
        sessionID = 0 // If there's an error, set sessionID to 0
    }

    var username string
    if sessionID > 0 {
        _ = database.Db.QueryRow("SELECT username FROM users WHERE id = ?", sessionID).Scan(&username)
    }

    postPageData := struct {
        Post      model.Post `json:"post"`
        SessionID int        `json:"sessionID"`
        Username  string     `json:"username"`
    }{
        Post:      post,
        SessionID: sessionID,
        Username:  username,
    }

    log.Printf("Successfully returning post data for ID %s", postID)
    util.ExecuteJSON(w, postPageData, http.StatusOK)
}