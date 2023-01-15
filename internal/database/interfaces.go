package database

import (
	"context"

	"chat-app/internal/models"
)

type UserRepository interface {
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateUser(ctx context.Context, req *models.RegisterRequest) (*models.User, error)
	GetUserByID(ctx context.Context, id int) (*models.User, error)
}

type RoomRepository interface {
	GetOrCreateRoom(ctx context.Context, name string) (int, error)
	CreateRoom(ctx context.Context, req *models.CreateRoomRequest, ownerID int) (*models.Room, error)
	GetRoomByID(ctx context.Context, id int) (*models.Room, error)
	ListUserRooms(ctx context.Context, userID int) ([]*models.Room, error)
	DeleteRoom(ctx context.Context, roomID, ownerID int) error
}

type MessageRepository interface {
	SaveMessage(ctx context.Context, userID, roomID int, content string) error
	LoadRecentMessages(ctx context.Context, roomID, limit int) ([]*models.Message, error)
}

type SessionRepository interface {
	CreateActiveSession(ctx context.Context, userID, roomID int, sessionID string) error
	RemoveActiveSession(ctx context.Context, userID, roomID int, sessionID string) error
	UpdateSessionActivity(ctx context.Context, userID, roomID int, sessionID string) error
	GetActiveUsersInRoom(ctx context.Context, roomID int) ([]*models.ActiveUser, error)
}

type MembershipRepository interface {
	AddMembership(ctx context.Context, userID, roomID int) error
	RemoveMembership(ctx context.Context, userID, roomID int) error
	IsMember(ctx context.Context, userID, roomID int) (bool, error)
	GetRoomMembers(ctx context.Context, roomID int) ([]*models.Member, error)
}

type Database interface {
	UserRepository
	RoomRepository
	MessageRepository
	SessionRepository
	MembershipRepository
	Close() error
}