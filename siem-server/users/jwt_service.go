package users

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const accessTokenTTL = 15 * time.Minute // короткоживущий access token

// JWTClaims — кастомные claims для JWT
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

// JWTService — сервис работы с JWT
type JWTService struct {
	secret []byte
}

func NewJWTService(jwtSecret string) (*JWTService, error) {
	if len(jwtSecret) == 0 {
		slog.Error("jwt secret is empty")
		return nil, fmt.Errorf("jwt secret lenth is 0")
	}
	return &JWTService{secret: []byte(jwtSecret)}, nil
}

// GenerateAccessToken — создаёт JWT access token (15 мин)
func (j *JWTService) GenerateAccessToken(user *User) (string, error) {
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
			Issuer:    "siem-server",
		},
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("sign token error: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken — проверяет и парсит JWT access token
func (j *JWTService) ValidateAccessToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ExpiresIn — возвращает время жизни access token в секундах
func ExpiresIn() int64 {
	return int64(accessTokenTTL.Seconds())
}
