package users

import (
	"errors"
	"fmt"
	"os"
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

func NewJWTService() *JWTService {
	secret := os.Getenv("JWT_SECRET")
	return &JWTService{secret: []byte(secret)}
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
		return "", fmt.Errorf("ошибка подписи токена: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken — проверяет и парсит JWT access token
func (j *JWTService) ValidateAccessToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", t.Header["alg"])
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
