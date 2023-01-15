package websocket

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"chat-app/internal/database"
	"chat-app/internal/models"
	"chat-app/pkg/logger"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    int
	username  string
	roomID    int
	sessionID string
	db        database.Database
}

func NewClient(hub *Hub, conn *websocket.Conn, userID int, username string, roomID int, db database.Database) (*Client, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	client := &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		userID:    userID,
		username:  username,
		roomID:    roomID,
		sessionID: sessionID,
		db:        db,
	}

	// Create active session in database
	ctx := context.Background()
	if err := db.CreateActiveSession(ctx, userID, roomID, sessionID); err != nil {
		logger.Error("Error creating active session: %v", err)
		return nil, fmt.Errorf("error creating session: %w", err)
	}

	return client, nil
}

func (c *Client) ReadPump() {
	defer func() {
		// Remove active session from database
		ctx := context.Background()
		if err := c.db.RemoveActiveSession(ctx, c.userID, c.roomID, c.sessionID); err != nil {
			logger.Error("Error removing active session: %v", err)
		}
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	// Set read deadline and pong handler for connection health
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket error: %v", err)
			}
			break
		}

		// Update session activity
		ctx := context.Background()
		if err := c.db.UpdateSessionActivity(ctx, c.userID, c.roomID, c.sessionID); err != nil {
			logger.Error("Error updating session activity: %v", err)
		}

		// Save message to database
		if err := c.db.SaveMessage(ctx, c.userID, c.roomID, string(message)); err != nil {
			logger.Error("Error saving message: %v", err)
		}

		// Create structured message for broadcast
		msgData := models.WebSocketMessage{
			Type:      models.MessageTypeMessage,
			Text:      string(message),
			Sender:    c.username,
			Timestamp: time.Now().Format(time.RFC3339),
		}

		if data, err := json.Marshal(msgData); err == nil {
			c.hub.Broadcast <- data
		} else {
			logger.Error("Error marshaling message: %v", err)
			// Fallback to simple text broadcast
			c.hub.Broadcast <- message
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.Error("Write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) SendRecentMessages() {
	ctx := context.Background()
	messages, err := c.db.LoadRecentMessages(ctx, c.roomID, 10)
	if err != nil {
		logger.Error("Error loading recent messages: %v", err)
		return
	}

	for _, msg := range messages {
		historyMsg := models.WebSocketMessage{
			Type:      models.MessageTypeMessage,
			Text:      fmt.Sprintf("%s: %s", msg.Username, msg.Content),
			Sender:    "system",
			Timestamp: msg.CreatedAt.Format(time.RFC3339),
		}

		if data, err := json.Marshal(historyMsg); err == nil {
			select {
			case c.send <- data:
			default:
				close(c.send)
				return
			}
		}
	}
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}