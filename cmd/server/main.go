package main

import (
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"chat-app/internal/auth"
	"chat-app/internal/config"
	"chat-app/internal/database"
	"chat-app/internal/handlers"
	"chat-app/internal/services"
	"chat-app/internal/websocket"
	"chat-app/pkg/logger"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database.URL)
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize services
	authService := auth.NewService(db, cfg)
	roomService := services.NewRoomService(db)

	// Initialize WebSocket hub manager
	hubManager := websocket.NewManager(db)

	// Initialize handlers
	authHandlers := handlers.NewAuthHandlers(authService)
	roomHandlers := handlers.NewRoomHandlers(roomService, authService)
	wsHandlers := handlers.NewWebSocketHandlers(authService, roomService, hubManager, db)

	// Setup routes
	mux := http.NewServeMux()
	setupRoutes(mux, authHandlers, roomHandlers, wsHandlers)

	// Create server
	server := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server
	logger.Info("🚀 Server started on http://localhost%s", cfg.Server.Port)
	logger.Info("📡 WebSocket endpoint: ws://localhost%s/ws", cfg.Server.Port)
	printAPIEndpoints()

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Server shutting down...")
}

func setupRoutes(mux *http.ServeMux, authHandlers *handlers.AuthHandlers, roomHandlers *handlers.RoomHandlers, wsHandlers *handlers.WebSocketHandlers) {
	// Auth routes
	mux.HandleFunc("/login", authHandlers.Login)
	mux.HandleFunc("/register", authHandlers.Register)

	// Room routes
	mux.HandleFunc("/rooms", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rooms" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		
		switch r.Method {
		case http.MethodGet:
			roomHandlers.ListRooms(w, r)
		case http.MethodPost:
			roomHandlers.CreateRoom(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Room sub-routes
	mux.HandleFunc("/rooms/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rooms" {
			http.Error(w, "use /rooms endpoint", http.StatusBadRequest)
			return
		}

		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 || parts[2] == "" {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		// /rooms/{id}/invite
		if len(parts) == 4 && parts[3] == "invite" && r.Method == http.MethodPost {
			roomHandlers.InviteUser(w, r)
			return
		}

		// /rooms/{id}/members
		if len(parts) == 4 && parts[3] == "members" && r.Method == http.MethodGet {
			roomHandlers.GetRoomMembers(w, r)
			return
		}

		// /rooms/{id}/leave
		if len(parts) == 4 && parts[3] == "leave" && r.Method == http.MethodDelete {
			roomHandlers.LeaveRoom(w, r)
			return
		}

		// /rooms/{id}/active
		if len(parts) == 4 && parts[3] == "active" && r.Method == http.MethodGet {
			roomHandlers.GetActiveUsers(w, r)
			return
		}

		// /rooms/{id} DELETE
		if len(parts) == 3 && r.Method == http.MethodDelete {
			roomHandlers.DeleteRoom(w, r)
			return
		}

		http.Error(w, "endpoint not found", http.StatusNotFound)
	})

	// WebSocket route
	mux.HandleFunc("/ws", wsHandlers.HandleWebSocket)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func printAPIEndpoints() {
	logger.Info("🔗 API endpoints:")
	logger.Info("   POST /login")
	logger.Info("   POST /register")
	logger.Info("   GET  /rooms")
	logger.Info("   POST /rooms")
	logger.Info("   GET  /rooms/{id}/members")
	logger.Info("   POST /rooms/{id}/invite")
	logger.Info("   DELETE /rooms/{id}/leave")
	logger.Info("   GET  /rooms/{id}/active")
	logger.Info("   DELETE /rooms/{id}")
}