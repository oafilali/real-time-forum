package post

import (
	"forum/internal/database"
	"forum/internal/model"
	"log"
)

func CreatePost(userID int, title, content, category string) error {
	tx, err := database.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO posts (user_id, title, content, category, date) VALUES (?, ?, ?, ?, datetime('now'))",
		userID, title, content, category,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func FetchPosts() ([]model.HomePageData, error) {
	postRows, err := database.Db.Query(`
		SELECT 
			p.id, 
			p.title, 
			p.content, 
			COALESCE(u.username, 'Unknown') AS username,
			COALESCE(SUM(CASE WHEN r.type = 'like' THEN 1 ELSE 0 END), 0) AS likes, 
			COALESCE(SUM(CASE WHEN r.type = 'dislike' THEN 1 ELSE 0 END), 0) AS dislikes,
			p.date
		FROM posts p
		LEFT JOIN users u ON p.user_id = u.id
		LEFT JOIN reactions r ON p.id = r.post_id AND r.comment_id IS NULL
		GROUP BY p.id, p.title, p.content, u.username, p.date;
	`)
	if err != nil {
		log.Println("Error fetching posts:", err)
		return []model.HomePageData{}, nil // Return empty slice instead of error
	}
	defer postRows.Close()

	var allPosts []model.HomePageData

	for postRows.Next() {
		var post model.HomePageData
		err := postRows.Scan(&post.ID, &post.Title, &post.Content, &post.Username, &post.Likes, &post.Dislikes, &post.Date)
		if err != nil {
			log.Println("Error scanning post row:", err)
			continue // Skip problematic rows instead of failing
		}
		allPosts = append(allPosts, post)
	}
	return allPosts, nil
}

func GetPostId() (id int, err error) {
	err = database.Db.QueryRow("SELECT last_insert_rowid()").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func FetchPost(postID string) (model.Post, error) {
	var post model.Post
	err := database.Db.QueryRow("SELECT id, user_id, title, content, category, date FROM posts WHERE id = ?", postID).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.Category, &post.Date,
	)
	if err != nil {
		return model.Post{}, err
	}
	username := ""
	err = database.Db.QueryRow("SELECT username FROM users WHERE id= ?", post.UserID).Scan(&username)
	if err != nil {
		username = "Unknown"
	}
	post.Username = username
	return post, err
}
