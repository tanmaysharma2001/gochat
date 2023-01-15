package services

import (
	"context"
	"fmt"

	"chat-app/internal/database"
	"chat-app/internal/models"
)

type RoomService struct {
	db database.Database
}

func NewRoomService(db database.Database) *RoomService {
	return &RoomService{db: db}
}

func (s *RoomService) CreateRoom(ctx context.Context, req *models.CreateRoomRequest, ownerID int) (*models.Room, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("room name is required")
	}

	return s.db.CreateRoom(ctx, req, ownerID)
}

func (s *RoomService) ListUserRooms(ctx context.Context, userID int) ([]*models.Room, error) {
	return s.db.ListUserRooms(ctx, userID)
}

func (s *RoomService) GetRoom(ctx context.Context, roomID int) (*models.Room, error) {
	return s.db.GetRoomByID(ctx, roomID)
}

func (s *RoomService) DeleteRoom(ctx context.Context, roomID, ownerID int) error {
	return s.db.DeleteRoom(ctx, roomID, ownerID)
}

func (s *RoomService) InviteUser(ctx context.Context, roomID, inviterID int, email string) error {
	// Get room to check permissions
	room, err := s.db.GetRoomByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("room not found")
	}

	// Check if inviter has permission
	if !room.IsPublic {
		canInvite := (room.OwnerID == inviterID)
		if !canInvite {
			isMember, err := s.db.IsMember(ctx, inviterID, roomID)
			if err != nil || !isMember {
				return fmt.Errorf("forbidden - not authorized to invite to this room")
			}
		}
	}

	// Get user by email
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Add membership
	return s.db.AddMembership(ctx, user.ID, roomID)
}

func (s *RoomService) LeaveRoom(ctx context.Context, userID, roomID int) error {
	isMember, err := s.db.IsMember(ctx, userID, roomID)
	if err != nil {
		return fmt.Errorf("database error")
	}
	if !isMember {
		return fmt.Errorf("not a member of this room")
	}

	return s.db.RemoveMembership(ctx, userID, roomID)
}

func (s *RoomService) GetRoomMembers(ctx context.Context, roomID, userID int) ([]*models.Member, error) {
	// Check access permissions
	room, err := s.db.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("room not found")
	}

	if !room.IsPublic {
		isMember, err := s.db.IsMember(ctx, userID, roomID)
		if err != nil || !isMember {
			return nil, fmt.Errorf("forbidden")
		}
	}

	return s.db.GetRoomMembers(ctx, roomID)
}

func (s *RoomService) GetActiveUsers(ctx context.Context, roomID, userID int) ([]*models.ActiveUser, error) {
	// Check access permissions
	room, err := s.db.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("room not found")
	}

	if !room.IsPublic {
		isMember, err := s.db.IsMember(ctx, userID, roomID)
		if err != nil || !isMember {
			return nil, fmt.Errorf("forbidden")
		}
	}

	return s.db.GetActiveUsersInRoom(ctx, roomID)
}

func (s *RoomService) CanUserAccessRoom(ctx context.Context, userID, roomID int) (bool, error) {
	room, err := s.db.GetRoomByID(ctx, roomID)
	if err != nil {
		return false, err
	}

	if room.IsPublic {
		return true, nil
	}

	return s.db.IsMember(ctx, userID, roomID)
}