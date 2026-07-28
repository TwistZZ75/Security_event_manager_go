package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost        = 12
	refreshTokenBytes = 32
	refreshTokenTTL   = 7 * 24 * time.Hour // 7 дней
)

type UserStorage struct {
	pool *pgxpool.Pool
}

func NewUserStorage(pool *pgxpool.Pool) *UserStorage {
	return &UserStorage{pool: pool}
}

// Create — создаёт нового пользователя с хешированием пароля
func (s *UserStorage) Create(ctx context.Context, input *RegisterInput) (*User, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid user data %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password error: %w", err)
	}

	query := `
		INSERT INTO users (username, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
		RETURNING id, username, email, password_hash, role, is_active, created_at, updated_at, last_login_at
	`

	row := s.pool.QueryRow(ctx, query,
		input.Username,
		input.Email,
		string(hash),
		input.Role,
	)

	user, err := s.scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("creating user error: %w", err)
	}
	return user, nil
}

// GetByID — получить пользователя по ID
func (s *UserStorage) GetByID(ctx context.Context, id int64) (*User, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	query := `
		SELECT id, username, email, password_hash, role, is_active, created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`
	return s.scanUser(s.pool.QueryRow(ctx, query, id))
}

// GetByUsername — получить пользователя по логину
func (s *UserStorage) GetByUsername(ctx context.Context, username string) (*User, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	query := `
		SELECT id, username, email, password_hash, role, is_active, created_at, updated_at, last_login_at
		FROM users WHERE username = $1
	`
	return s.scanUser(s.pool.QueryRow(ctx, query, username))
}

// List — список всех пользователей (только для admin)
func (s *UserStorage) List(ctx context.Context) ([]*User, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	query := `
		SELECT id, username, email, password_hash, role, is_active, created_at, updated_at, last_login_at
		FROM users ORDER BY created_at DESC
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get user info error: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user, err := s.scanUser(rows)
		if err != nil {
			slog.Warn("Failed to scan user row", "error", err)
			continue
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// Update — обновить данные пользователя
func (s *UserStorage) Update(ctx context.Context, id int64, input *UpdateUserInput) (*User, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	pos := 1

	if input.Email != "" {
		setParts = append(setParts, fmt.Sprintf("email = $%d", pos))
		args = append(args, input.Email)
		pos++
	}
	if input.Password != "" {
		if len(input.Password) < 8 {
			return nil, ErrPasswordTooShort
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcryptCost)
		if err != nil {
			return nil, fmt.Errorf("hash password error: %w", err)
		}
		setParts = append(setParts, fmt.Sprintf("password_hash = $%d", pos))
		args = append(args, string(hash))
		pos++
	}
	if input.Role != "" {
		setParts = append(setParts, fmt.Sprintf("role = $%d", pos))
		args = append(args, input.Role)
		pos++
	}
	if input.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", pos))
		args = append(args, *input.IsActive)
		pos++
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE users SET %s WHERE id = $%d
		RETURNING id, username, email, password_hash, role, is_active, created_at, updated_at, last_login_at
	`, strings.Join(setParts, ", "), pos)

	return s.scanUser(s.pool.QueryRow(ctx, query, args...))
}

// Delete — удалить пользователя
func (s *UserStorage) Delete(ctx context.Context, id int64) error {
	if s.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	result, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Authenticate — проверяет логин/пароль, обновляет last_login_at
func (s *UserStorage) Authenticate(ctx context.Context, username, password string) (*User, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	user, err := s.GetByUsername(ctx, username)
	if err != nil {
		// timing-safe: считаем хеш даже если пользователь не найден
		_ = bcrypt.CompareHashAndPassword(
			[]byte("$2a$12$dummydummydummydummydunmydummy00"),
			[]byte(password),
		)
		return nil, ErrInvalidPassword
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidPassword
	}

	// обновляем время последнего входа
	if _, err = s.pool.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID); err != nil {
		slog.Warn("Failed to update last login time", "user_id", user.ID, "error", err)
	}

	return user, nil
}

// StoreRefreshToken — сохраняет refresh token (один на пользователя)
func (s *UserStorage) StoreRefreshToken(ctx context.Context, userID int64, token string) error {
	if s.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	expiresAt := time.Now().Add(refreshTokenTTL)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE SET 
			token = EXCLUDED.token,
			expires_at = EXCLUDED.expires_at,
			created_at = NOW()
	`, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("refresh token store error %w", err)
	}
	return nil
}

// ValidateRefreshToken — проверяет refresh token и возвращает пользователя
func (s *UserStorage) ValidateRefreshToken(ctx context.Context, token string) (*User, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.role, u.is_active,
		       u.created_at, u.updated_at, u.last_login_at
		FROM users u
		JOIN refresh_tokens rt ON rt.user_id = u.id
		WHERE rt.token = $1 AND rt.expires_at > NOW()
	`
	user, err := s.scanUser(s.pool.QueryRow(ctx, query, token))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("scan user error %w", err)
	}
	if !user.IsActive {
		return nil, ErrUserInactive
	}
	return user, nil
}

// RevokeRefreshToken — отзывает refresh token (выход)
func (s *UserStorage) RevokeRefreshToken(ctx context.Context, token string) error {
	if s.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token = $1`, token); err != nil {
		return fmt.Errorf("deleting ref token by token id error %w", err)
	}
	return nil
}

// RevokeAllUserTokens — отзывает все токены пользователя
func (s *UserStorage) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	if s.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting ref token by user id %w", err)
	}
	return nil
}

// GenerateRefreshToken — генерирует криптографически случайный токен
func GenerateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token error: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (s *UserStorage) scanUser(scanner interface {
	Scan(dest ...interface{}) error
}) (*User, error) {
	var user User
	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user error: %w", err)
	}
	return &user, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unique")
}
