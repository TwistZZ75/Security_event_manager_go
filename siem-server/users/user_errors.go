package users

import "errors"

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user with that email already exist")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrUserInactive     = errors.New("account was deactivate")
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("expired token")
	ErrInvalidUsername  = errors.New("usermane must contain 3-50 symbols")
	ErrPasswordTooShort = errors.New("password too short. password must contain 8 symbols")
	ErrInvalidEmail     = errors.New("invalid email")
	ErrInvalidRole      = errors.New("invalid role")
	ErrAccessDenied     = errors.New("access denied")
)
