package users

import "errors"

var (
	ErrUserNotFound     = errors.New("пользователь не найден")
	ErrUserExists       = errors.New("пользователь с таким именем или email уже существует")
	ErrInvalidPassword  = errors.New("неверный пароль")
	ErrUserInactive     = errors.New("аккаунт деактивирован")
	ErrInvalidToken     = errors.New("недействительный токен")
	ErrTokenExpired     = errors.New("токен истёк")
	ErrInvalidUsername  = errors.New("имя пользователя должно быть от 3 до 50 символов")
	ErrPasswordTooShort = errors.New("пароль должен содержать минимум 8 символов")
	ErrInvalidEmail     = errors.New("некорректный email")
	ErrInvalidRole      = errors.New("недопустимая роль")
	ErrAccessDenied     = errors.New("недостаточно прав")
)
