package model

type Session struct {
	Sid     string `json:"sid"`
	Expires int64  `json:"expires"`
}

type Response struct {
	Session *Session    `json:"session"`
	Data    interface{} `json:"data"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserDetails struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Avatar    string `json:"avatar"`
	Online    bool   `json:"online"`
}

type UserDetailsEntity struct {
	IDs      []string               `json:"ids"`
	Entities map[string]UserDetails `json:"entities"`
}

type File struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	Downloading bool   `json:"downloading"`
	Progress    int8   `json:"progress"`
	Url         string `json:"url"`
	Error       bool   `json:"error"`
}

type FileEntity struct {
	IDs      []string        `json:"ids"`
	Entities map[string]File `json:"entities"`
}
