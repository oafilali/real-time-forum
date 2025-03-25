package model

type Post struct {
	ID       int
	Username string
	UserID   int
	Title    string
	Content  string
	Category string
	Likes    int
	Dislikes int
	Comments []Comment
	Date     string
}

type Comment struct {
	ID       int
	Username string
	UserID   int
	Content  string
	Likes    int
	Dislikes int
}

type PostPageData struct {
	Post      Post
	SessionID int
}

type HomePageData struct {
	ID       int
	Title    string
	Content  string
	Username string
	Likes    int
	Dislikes int
	Date     string
}

type Data struct {
	Posts     []HomePageData
	SessionID int
	Username  string
}

type MsgData struct {
	Message string `json:"message"`
}

type PostData struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
}
