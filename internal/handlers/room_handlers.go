package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"chat-app/internal/auth"
	"chat-app/internal/models"
	"chat-app/internal/services"
	"chat-app/pkg/logger"
)

type RoomHandlers struct {
	roomService *services.RoomService
	authService *auth.Service
}

func NewRoomHandlers(roomService *services.RoomService, authService *auth.Service) *RoomHandlers {
	return &RoomHandlers{
		roomService: roomService,
		authService: authService,
	}
}

func (h *RoomHandlers) CreateRoom(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	room, err := h.roomService.CreateRoom(r.Context(), &req, user.ID)
	if err != nil {
		logger.Error("Create room error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(room)
}

func (h *RoomHandlers) ListRooms(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rooms, err := h.roomService.ListUserRooms(r.Context(), user.ID)
	if err != nil {
		logger.Error("List rooms error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rooms)
}

func (h *RoomHandlers) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID, err := h.getRoomIDFromPath(r)
	if err != nil {
		http.Error(w, "invalid room ID", http.StatusBadRequest)
		return
	}

	if err := h.roomService.DeleteRoom(r.Context(), roomID, user.ID); err != nil {
		logger.Error("Delete room error: %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("room deleted successfully"))
}

func (h *RoomHandlers) InviteUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID, err := h.getRoomIDFromPath(r)
	if err != nil {
		http.Error(w, "invalid room ID", http.StatusBadRequest)
		return
	}

	var req models.InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.roomService.InviteUser(r.Context(), roomID, user.ID, req.Email); err != nil {
		logger.Error("Invite user error: %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("user invited to room"))
}

func (h *RoomHandlers) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID, err := h.getRoomIDFromPath(r)
	if err != nil {
		http.Error(w, "invalid room ID", http.StatusBadRequest)
		return
	}

	if err := h.roomService.LeaveRoom(r.Context(), user.ID, roomID); err != nil {
		logger.Error("Leave room error: %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("left room successfully"))
}

func (h *RoomHandlers) GetRoomMembers(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID, err := h.getRoomIDFromPath(r)
	if err != nil {
		http.Error(w, "invalid room ID", http.StatusBadRequest)
		return
	}

	members, err := h.roomService.GetRoomMembers(r.Context(), roomID, user.ID)
	if err != nil {
		logger.Error("Get room members error: %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (h *RoomHandlers) GetActiveUsers(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromToken(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID, err := h.getRoomIDFromPath(r)
	if err != nil {
		http.Error(w, "invalid room ID", http.StatusBadRequest)
		return
	}

	activeUsers, err := h.roomService.GetActiveUsers(r.Context(), roomID, user.ID)
	if err != nil {
		logger.Error("Get active users error: %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"room_id":      roomID,
		"active_users": activeUsers,
		"count":        len(activeUsers),
	})
}

func (h *RoomHandlers) getUserFromToken(r *http.Request) (*models.User, error) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		return nil, fmt.Errorf("missing token")
	}

	return h.authService.GetUserFromToken(r.Context(), tokenStr)
}

func (h *RoomHandlers) getRoomIDFromPath(r *http.Request) (int, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		return 0, fmt.Errorf("invalid path")
	}
	
	return strconv.Atoi(parts[2])
}