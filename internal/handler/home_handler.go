package handler

import (
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
	
	if len(allPosts) > 5 {
		allPosts = allPosts[:5]
	}

	data := model.Data{
		Posts:     allPosts,
		SessionID: userID,
		Username:  username,
	}

	// Return JSON response
	util.ExecuteJSON(w, data, http.StatusOK)
}