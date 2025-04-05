package handler

import (
	"forum/internal/database"
	"forum/internal/model"
	"forum/internal/util"
	"log"
	"net/http"
)

// GetAllUsersHandler returns a list of all registered users
func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		util.ExecuteJSON(w, model.MsgData{"Invalid request method"}, http.StatusMethodNotAllowed)
		return
	}

	rows, err := database.Db.Query("SELECT id, username FROM users")
	if err != nil {
		log.Println("Error fetching users:", err)
		util.ExecuteJSON(w, model.MsgData{"Failed to load users"}, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserInfo struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}

	var users []UserInfo

	for rows.Next() {
		var user UserInfo
		if err := rows.Scan(&user.ID, &user.Username); err != nil {
			log.Println("Error scanning user row:", err)
			continue
		}
		users = append(users, user)
	}

	// Return the user list
	util.ExecuteJSON(w, struct {
		Users []UserInfo `json:"users"`
	}{
		Users: users,
	}, http.StatusOK)
}