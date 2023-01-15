package models

import "time"

type Room struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	IsPublic  bool      `json:"is_public"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	RoomID    int       `json:"room_id"`
	Content   string    `json:"content"`
	Username  string    `json:"username,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ActiveSession struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	RoomID      int       `json:"room_id"`
	SessionID   string    `json:"session_id"`
	ConnectedAt time.Time `json:"connected_at"`
	LastSeen    time.Time `json:"last_seen"`
}

type ActiveUser struct {
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	ConnectedAt time.Time `json:"connected_at"`
	LastSeen    time.Time `json:"last_seen"`
	Status      string    `json:"status"`
}

type CreateRoomRequest struct {
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
}

type InviteRequest struct {
	Email string `json:"email"`
}

type Member struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}