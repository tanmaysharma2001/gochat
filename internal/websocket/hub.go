package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"chat-app/internal/database"
	"chat-app/internal/models"
	"chat-app/pkg/logger"
)

type Hub struct {
	clients       map[*Client]bool
	Broadcast     chan []byte
	Register      chan *Client
	Unregister    chan *Client
	roomID        int
	onlineUsers   map[string]bool
	shutdown      chan bool
	lastActivity  time.Time
	db            database.Database
}

func NewHub(roomID int, db database.Database) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		Broadcast:     make(chan []byte),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		roomID:        roomID,
		onlineUsers:   make(map[string]bool),
		shutdown:      make(chan bool),
		lastActivity:  time.Now(),
		db:            db,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.shutdown:
			for client := range h.clients {
				close(client.send)
			}
			return

		case client := <-h.Register:
			h.clients[client] = true
			h.lastActivity = time.Now()
			h.onlineUsers[client.username] = true
			h.broadcastPresenceUpdate()
			logger.Info("User %s joined room %d", client.username, h.roomID)

		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				delete(h.onlineUsers, client.username)
				h.broadcastPresenceUpdate()
				logger.Info("User %s left room %d", client.username, h.roomID)
			}

		case message := <-h.Broadcast:
			h.lastActivity = time.Now()
			h.broadcastToAll(message)
		}
	}
}

func (h *Hub) broadcastToAll(message []byte) {
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
			delete(h.onlineUsers, client.username)
		}
	}
}

func (h *Hub) broadcastPresenceUpdate() {
	ctx := context.Background()
	activeUsers, err := h.db.GetActiveUsersInRoom(ctx, h.roomID)
	if err != nil {
		logger.Error("Error getting active users for presence update: %v", err)
		return
	}

	presenceMsg := models.WebSocketMessage{
		Type:        models.MessageTypePresenceUpdate,
		ActiveUsers: activeUsers,
		UserCount:   len(activeUsers),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if data, err := json.Marshal(presenceMsg); err == nil {
		h.broadcastToAll(data)
	} else {
		logger.Error("Error marshaling presence update: %v", err)
	}
}

func (h *Hub) GetOnlineUserCount() int {
	return len(h.onlineUsers)
}

func (h *Hub) ShutdownHub() {
	select {
	case h.shutdown <- true:
	default:
	}
}

func (h *Hub) StartCleanupRoutine() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if time.Since(h.lastActivity) > 30*time.Minute && len(h.clients) == 0 {
			h.ShutdownHub()
			return
		}
	}
}

// Hub Manager
type Manager struct {
	hubs   map[int]*Hub
	mutex  sync.Mutex
	db     database.Database
}

func NewManager(db database.Database) *Manager {
	manager := &Manager{
		hubs: make(map[int]*Hub),
		db:   db,
	}
	
	go manager.cleanupUnusedHubs()
	return manager
}

func (m *Manager) GetHubForRoom(roomID int) *Hub {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	hub, exists := m.hubs[roomID]
	if !exists {
		hub = NewHub(roomID, m.db)
		m.hubs[roomID] = hub
		go hub.Run()
		go hub.StartCleanupRoutine()
	}
	return hub
}

func (m *Manager) cleanupUnusedHubs() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mutex.Lock()
		for roomID, hub := range m.hubs {
			if hub.GetOnlineUserCount() == 0 {
				hub.ShutdownHub()
				delete(m.hubs, roomID)
				logger.Debug("Cleaned up unused hub for room %d", roomID)
			}
		}
		m.mutex.Unlock()
	}
}