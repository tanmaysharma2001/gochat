package auth

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"chat-app/internal/config"
	"chat-app/internal/database"
	"chat-app/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db  database.Database
	cfg *config.Config
}

func NewService(db database.Database, cfg *config.Config) *Service {
	return &Service{
		db:  db,
		cfg: cfg,
	}
}

func (s *Service) Register(ctx context.Context, req *models.RegisterRequest) (*models.LoginResponse, error) {
	// Validate input
	if err := s.validateRegistrationRequest(req); err != nil {
		return nil, err
	}

	// Create user
	user, err := s.db.CreateUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *Service) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	// Get user by email
	user, err := s.db.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Remove sensitive data
	user.PasswordHash = ""

	return &models.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *Service) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.cfg.JWT.Secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *Service) GetUserFromToken(ctx context.Context, tokenString string) (*models.User, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	userIDFloat, ok := (*claims)["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	userID := int(userIDFloat)
	return s.db.GetUserByID(ctx, userID)
}

func (s *Service) generateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(s.cfg.JWT.ExpiresIn).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.cfg.JWT.Secret)
}

func (s *Service) validateRegistrationRequest(req *models.RegisterRequest) error {
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return fmt.Errorf("missing required fields")
	}

	// Validate email format
	if !isValidEmail(req.Email) {
		return fmt.Errorf("invalid email format")
	}

	// Validate password strength
	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Sanitize and validate username
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 30 {
		return fmt.Errorf("username must be 3-30 characters long")
	}

	return nil
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}