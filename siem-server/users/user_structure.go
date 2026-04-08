package users

import "time"

// Role — роль пользователя в системе
type Role string

const (
	RoleAdmin   Role = "admin"   // Полный доступ: управление правилами, алертами, пользователями
	RoleAnalyst Role = "analyst" // Создание/редактирование правил, работа с алертами
	RoleViewer  Role = "viewer"  // Только чтение
)

// Permissions — список разрешений для каждой роли
var Permissions = map[Role]map[string]bool{
	RoleAdmin: {
		"users:read": true, "users:write": true, "users:delete": true,
		"rules:read": true, "rules:write": true, "rules:delete": true,
		"alerts:read": true, "alerts:write": true,
		"actions:read": true,
		"agents:read":  true,
		"events:read":  true,
	},
	RoleAnalyst: {
		"rules:read": true, "rules:write": true,
		"alerts:read": true, "alerts:write": true,
		"actions:read": true,
		"agents:read":  true,
		"events:read":  true,
	},
	RoleViewer: {
		"rules:read":   true,
		"alerts:read":  true,
		"actions:read": true,
		"agents:read":  true,
		"events:read":  true,
	},
}

// HasPermission проверяет, есть ли у роли нужное разрешение
func (r Role) HasPermission(perm string) bool {
	if perms, ok := Permissions[r]; ok {
		return perms[perm]
	}
	return false
}

// User — основная модель пользователя
type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         Role       `json:"role"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// SafeUser — пользователь без чувствительных полей (для API-ответов)
type SafeUser struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Role        Role       `json:"role"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

func (u *User) ToSafe() *SafeUser {
	return &SafeUser{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Role:        u.Role,
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		LastLoginAt: u.LastLoginAt,
	}
}

// DTO для запросов

type RegisterInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role,omitempty"` // если не указана — viewer
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateUserInput struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Role     Role   `json:"role,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

// TokenPair — пара токенов после успешного входа
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"` // секунды до истечения access token
	User         *SafeUser `json:"user"`
}

// Claims — payload JWT-токена
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

// Validate — базовая валидация входных данных при регистрации
func (r *RegisterInput) Validate() error {
	if len(r.Username) < 3 || len(r.Username) > 50 {
		return ErrInvalidUsername
	}
	if len(r.Password) < 8 {
		return ErrPasswordTooShort
	}
	if r.Email == "" {
		return ErrInvalidEmail
	}
	if r.Role == "" {
		r.Role = RoleViewer
	}
	// проверяем что роль валидна
	if r.Role != RoleAdmin && r.Role != RoleAnalyst && r.Role != RoleViewer {
		return ErrInvalidRole
	}
	return nil
}
