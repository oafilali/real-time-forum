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
	Likes    int
	Dislikes int
}

type Data struct {
	Posts     []HomePageData
	SessionID int
	Username  string
}
