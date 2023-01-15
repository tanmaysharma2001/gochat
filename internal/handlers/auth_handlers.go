package handlers

import (
	"encoding/json"
	"net/http"

	"chat-app/internal/auth"
	"chat-app/internal/models"
	"chat-app/pkg/logger"
)

type AuthHandlers struct {
	authService *auth.Service
}

func NewAuthHandlers(authService *auth.Service) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
	}
}

func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	response, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		logger.Error("Registration error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	response, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		logger.Error("Login error: %v", err)
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}