package database

import (
	"context"
	"fmt"

	"chat-app/internal/models"
	"chat-app/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type PostgresDB struct {
	pool *pgxpool.Pool
}

func NewPostgresDB(databaseURL string) (*PostgresDB, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to database successfully")
	return &PostgresDB{pool: pool}, nil
}

func (db *PostgresDB) Close() error {
	db.pool.Close()
	return nil
}

// User Repository Implementation
func (db *PostgresDB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, created_at FROM users WHERE email = $1`
	
	user := &models.User{}
	err := db.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

func (db *PostgresDB) CreateUser(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		INSERT INTO users (username, email, password_hash, created_at) 
		VALUES ($1, $2, $3, NOW()) 
		RETURNING id, username, email, created_at`
	
	user := &models.User{PasswordHash: string(hash)}
	err = db.pool.QueryRow(ctx, query, req.Username, req.Email, string(hash)).Scan(
		&user.ID, &user.Username, &user.Email, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return user, nil
}

func (db *PostgresDB) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT id, username, email, created_at FROM users WHERE id = $1`
	
	user := &models.User{}
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

// Room Repository Implementation
func (db *PostgresDB) GetOrCreateRoom(ctx context.Context, name string) (int, error) {
	query := `
		INSERT INTO rooms (name, is_public, created_at) VALUES ($1, true, NOW())
		ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name
		RETURNING id`
	
	var roomID int
	err := db.pool.QueryRow(ctx, query, name).Scan(&roomID)
	return roomID, err
}

func (db *PostgresDB) CreateRoom(ctx context.Context, req *models.CreateRoomRequest, ownerID int) (*models.Room, error) {
	query := `
		INSERT INTO rooms (name, is_public, owner_id, created_at) 
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (name) DO UPDATE SET is_public = EXCLUDED.is_public
		RETURNING id, name, is_public, owner_id, created_at`
	
	room := &models.Room{}
	err := db.pool.QueryRow(ctx, query, req.Name, req.IsPublic, ownerID).Scan(
		&room.ID, &room.Name, &room.IsPublic, &room.OwnerID, &room.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}
	
	return room, nil
}

func (db *PostgresDB) GetRoomByID(ctx context.Context, id int) (*models.Room, error) {
	query := `SELECT id, name, is_public, owner_id, created_at FROM rooms WHERE id = $1`
	
	room := &models.Room{}
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&room.ID, &room.Name, &room.IsPublic, &room.OwnerID, &room.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	return room, nil
}

func (db *PostgresDB) ListUserRooms(ctx context.Context, userID int) ([]*models.Room, error) {
	query := `
		SELECT r.id, r.name, r.is_public, r.owner_id, r.created_at
		FROM rooms r
		LEFT JOIN memberships m ON r.id = m.room_id AND m.user_id = $1
		WHERE r.is_public = true OR m.user_id IS NOT NULL
		ORDER BY r.name`
	
	rows, err := db.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		room := &models.Room{}
		if err := rows.Scan(&room.ID, &room.Name, &room.IsPublic, &room.OwnerID, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	
	return rooms, nil
}

func (db *PostgresDB) DeleteRoom(ctx context.Context, roomID, ownerID int) error {
	// Check ownership first
	var currentOwnerID int
	err := db.pool.QueryRow(ctx, "SELECT owner_id FROM rooms WHERE id = $1", roomID).Scan(&currentOwnerID)
	if err != nil {
		return fmt.Errorf("room not found: %w", err)
	}
	
	if currentOwnerID != ownerID {
		return fmt.Errorf("forbidden - not the room owner")
	}

	// Delete in transaction
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete memberships
	if _, err := tx.Exec(ctx, "DELETE FROM memberships WHERE room_id = $1", roomID); err != nil {
		return err
	}
	
	// Delete messages
	if _, err := tx.Exec(ctx, "DELETE FROM messages WHERE room_id = $1", roomID); err != nil {
		return err
	}
	
	// Delete active sessions
	if _, err := tx.Exec(ctx, "DELETE FROM active_sessions WHERE room_id = $1", roomID); err != nil {
		return err
	}
	
	// Delete room
	if _, err := tx.Exec(ctx, "DELETE FROM rooms WHERE id = $1", roomID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Message Repository Implementation
func (db *PostgresDB) SaveMessage(ctx context.Context, userID, roomID int, content string) error {
	query := `INSERT INTO messages (user_id, room_id, content, created_at) VALUES ($1, $2, $3, NOW())`
	_, err := db.pool.Exec(ctx, query, userID, roomID, content)
	return err
}

func (db *PostgresDB) LoadRecentMessages(ctx context.Context, roomID, limit int) ([]*models.Message, error) {
	query := `
		SELECT m.id, m.user_id, m.room_id, m.content, u.username, m.created_at
		FROM messages m 
		JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1 
		ORDER BY m.created_at DESC 
		LIMIT $2`
	
	rows, err := db.pool.Query(ctx, query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		msg := &models.Message{}
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.RoomID, &msg.Content, &msg.Username, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	
	// Reverse to show oldest first
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	
	return messages, nil
}

// Session Repository Implementation
func (db *PostgresDB) CreateActiveSession(ctx context.Context, userID, roomID int, sessionID string) error {
	query := `
		INSERT INTO active_sessions (user_id, room_id, session_id, connected_at, last_seen) 
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, room_id, session_id) 
		DO UPDATE SET last_seen = NOW()`
	
	_, err := db.pool.Exec(ctx, query, userID, roomID, sessionID)
	return err
}

func (db *PostgresDB) RemoveActiveSession(ctx context.Context, userID, roomID int, sessionID string) error {
	query := `DELETE FROM active_sessions WHERE user_id = $1 AND room_id = $2 AND session_id = $3`
	_, err := db.pool.Exec(ctx, query, userID, roomID, sessionID)
	return err
}

func (db *PostgresDB) UpdateSessionActivity(ctx context.Context, userID, roomID int, sessionID string) error {
	query := `UPDATE active_sessions SET last_seen = NOW() WHERE user_id = $1 AND room_id = $2 AND session_id = $3`
	_, err := db.pool.Exec(ctx, query, userID, roomID, sessionID)
	return err
}

func (db *PostgresDB) GetActiveUsersInRoom(ctx context.Context, roomID int) ([]*models.ActiveUser, error) {
	// Clean up stale sessions
	cleanupQuery := `DELETE FROM active_sessions WHERE last_seen < NOW() - INTERVAL '5 minutes'`
	if _, err := db.pool.Exec(ctx, cleanupQuery); err != nil {
		logger.Error("Error cleaning stale sessions: %v", err)
	}

	query := `
		SELECT DISTINCT u.id, u.username, u.email, s.connected_at, s.last_seen
		FROM active_sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.room_id = $1
		ORDER BY u.username`
	
	rows, err := db.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activeUsers []*models.ActiveUser
	for rows.Next() {
		user := &models.ActiveUser{Status: "online"}
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.ConnectedAt, &user.LastSeen); err != nil {
			return nil, err
		}
		activeUsers = append(activeUsers, user)
	}
	
	return activeUsers, nil
}

// Membership Repository Implementation
func (db *PostgresDB) AddMembership(ctx context.Context, userID, roomID int) error {
	query := `
		INSERT INTO memberships (user_id, room_id) VALUES ($1, $2)
		ON CONFLICT (user_id, room_id) DO NOTHING`
	
	_, err := db.pool.Exec(ctx, query, userID, roomID)
	return err
}

func (db *PostgresDB) RemoveMembership(ctx context.Context, userID, roomID int) error {
	query := `DELETE FROM memberships WHERE user_id = $1 AND room_id = $2`
	_, err := db.pool.Exec(ctx, query, userID, roomID)
	return err
}

func (db *PostgresDB) IsMember(ctx context.Context, userID, roomID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM memberships WHERE user_id = $1 AND room_id = $2)`
	
	var exists bool
	err := db.pool.QueryRow(ctx, query, userID, roomID).Scan(&exists)
	return exists, err
}

func (db *PostgresDB) GetRoomMembers(ctx context.Context, roomID int) ([]*models.Member, error) {
	query := `
		SELECT u.id, u.username, u.email
		FROM memberships m
		JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1
		ORDER BY u.username`
	
	rows, err := db.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.Member
	for rows.Next() {
		member := &models.Member{}
		if err := rows.Scan(&member.ID, &member.Username, &member.Email); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	
	return members, nil
}