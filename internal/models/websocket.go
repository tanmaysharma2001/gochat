package models

type MessageType string

const (
	MessageTypeMessage        MessageType = "message"
	MessageTypeUserJoined     MessageType = "user_joined"
	MessageTypeUserLeft       MessageType = "user_left"
	MessageTypeOnlineUsers    MessageType = "online_users"
	MessageTypePresenceUpdate MessageType = "presence_update"
)

type WebSocketMessage struct {
	Type        MessageType   `json:"type"`
	Text        string        `json:"text,omitempty"`
	Sender      string        `json:"sender,omitempty"`
	Username    string        `json:"username,omitempty"`
	Timestamp   string        `json:"timestamp,omitempty"`
	Users       []string      `json:"users,omitempty"`
	ActiveUsers []*ActiveUser `json:"active_users,omitempty"`
	UserCount   int           `json:"user_count,omitempty"`
}