package handlers

import (
	"net/http"

	"chat-app/internal/auth"
	"chat-app/internal/database"
	"chat-app/internal/services"
	ws "chat-app/internal/websocket"
	"chat-app/pkg/logger"

	"github.com/gorilla/websocket"
)

type WebSocketHandlers struct {
	authService *auth.Service
	roomService *services.RoomService
	hubManager  *ws.Manager
	db          database.Database
	upgrader    websocket.Upgrader
}

func NewWebSocketHandlers(authService *auth.Service, roomService *services.RoomService, hubManager *ws.Manager, db database.Database) *WebSocketHandlers {
	return &WebSocketHandlers{
		authService: authService,
		roomService: roomService,
		hubManager:  hubManager,
		db:          db,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // Configure for production
		},
	}
}

func (h *WebSocketHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get JWT token from query parameters
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	// Validate token and get user
	user, err := h.authService.GetUserFromToken(r.Context(), tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Get room from query parameter
	roomName := r.URL.Query().Get("room")
	if roomName == "" {
		roomName = "general"
	}

	// Get or create room
	roomID, err := h.db.GetOrCreateRoom(r.Context(), roomName)
	if err != nil {
		logger.Error("Error creating room: %v", err)
		http.Error(w, "error accessing room", http.StatusInternalServerError)
		return
	}

	// Check if user can access room
	canAccess, err := h.roomService.CanUserAccessRoom(r.Context(), user.ID, roomID)
	if err != nil {
		http.Error(w, "error checking room access", http.StatusInternalServerError)
		return
	}
	if !canAccess {
		http.Error(w, "not a member of this room", http.StatusForbidden)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Upgrade error: %v", err)
		return
	}

	// Get hub for room
	hub := h.hubManager.GetHubForRoom(roomID)

	// Create client
	client, err := ws.NewClient(hub, conn, user.ID, user.Username, roomID, h.db)
	if err != nil {
		logger.Error("Error creating client: %v", err)
		conn.Close()
		return
	}

	// Register client with hub
	hub.Register <- client

	// Send recent messages to the new client
	go client.SendRecentMessages()

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()
}