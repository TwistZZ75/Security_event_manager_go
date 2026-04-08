package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Handler — HTTP-хендлеры для работы с пользователями и аутентификацией
type Handler struct {
	storage *UserStorage
	jwt     *JWTService
}

func NewHandler(storage *UserStorage, jwt *JWTService) *Handler {
	return &Handler{storage: storage, jwt: jwt}
}

// RegisterRoutes — регистрирует маршруты (вызвать в web_server/routes.go)
//
//	authRequired — middleware AuthMiddleware
//	adminOnly    — middleware RequireRole(RoleAdmin)
func (h *Handler) RegisterRoutes(
	api *mux.Router,
	authRequired func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
) {
	// Публичные маршруты
	api.Handle("/auth/register", h.wrapPublic(h.Register)).Methods("POST", "OPTIONS")
	api.Handle("/auth/login", h.wrapPublic(h.Login)).Methods("POST", "OPTIONS")
	api.Handle("/auth/refresh", h.wrapPublic(h.Refresh)).Methods("POST", "OPTIONS")

	// Требуют валидного токена
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authRequired)
	protected.HandleFunc("/auth/logout", h.Logout).Methods("POST", "OPTIONS")
	protected.HandleFunc("/auth/me", h.Me).Methods("GET", "OPTIONS")

	// Только admin
	admin := protected.PathPrefix("").Subrouter()
	admin.Use(adminOnly)
	admin.HandleFunc("/users", h.ListUsers).Methods("GET", "OPTIONS")
	admin.HandleFunc("/users/{id}", h.GetUser).Methods("GET", "OPTIONS")
	admin.HandleFunc("/users/{id}", h.UpdateUser).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/users/{id}", h.DeleteUser).Methods("DELETE", "OPTIONS")
}

// Register — POST /api/auth/register
// Первый пользователь автоматически получает роль admin
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "ошибка разбора запроса")
		return
	}

	// Первый пользователь — всегда admin
	ctx := r.Context()
	existing, _ := h.storage.List(ctx)
	if len(existing) == 0 {
		input.Role = RoleAdmin
	}

	user, err := h.storage.Create(ctx, &input)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserExists):
			respondError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrInvalidUsername), errors.Is(err, ErrPasswordTooShort),
			errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidRole):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "ошибка регистрации")
		}
		return
	}

	tokens, err := h.issueTokens(ctx, user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ошибка создания токенов")
		return
	}

	respondJSON(w, http.StatusCreated, tokens)
}

// Login — POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "ошибка разбора запроса")
		return
	}

	if input.Username == "" || input.Password == "" {
		respondError(w, http.StatusBadRequest, "логин и пароль обязательны")
		return
	}

	ctx := r.Context()
	user, err := h.storage.Authenticate(ctx, input.Username, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPassword):
			respondError(w, http.StatusUnauthorized, "неверный логин или пароль")
		case errors.Is(err, ErrUserInactive):
			respondError(w, http.StatusForbidden, "аккаунт деактивирован")
		default:
			respondError(w, http.StatusInternalServerError, "ошибка аутентификации")
		}
		return
	}

	tokens, err := h.issueTokens(ctx, user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ошибка создания токенов")
		return
	}

	respondJSON(w, http.StatusOK, tokens)
}

// Refresh — POST /api/auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input RefreshInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "ошибка разбора запроса")
		return
	}
	if input.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "refresh_token обязателен")
		return
	}

	ctx := r.Context()
	user, err := h.storage.ValidateRefreshToken(ctx, input.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "недействительный или истёкший refresh token")
		return
	}

	// Отзываем старый токен и выдаём новую пару
	_ = h.storage.RevokeRefreshToken(ctx, input.RefreshToken)

	tokens, err := h.issueTokens(ctx, user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ошибка создания токенов")
		return
	}

	respondJSON(w, http.StatusOK, tokens)
}

// Logout — POST /api/auth/logout (требует auth)
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var input RefreshInput
	if err := json.NewDecoder(r.Body).Decode(&input); err == nil && input.RefreshToken != "" {
		_ = h.storage.RevokeRefreshToken(r.Context(), input.RefreshToken)
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "выход выполнен"})
}

// Me — GET /api/auth/me (требует auth)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	// userID кладётся в контекст через AuthMiddleware
	userIDVal := r.Context().Value(struct{ key string }{"userID"})
	if userIDVal == nil {
		respondError(w, http.StatusUnauthorized, "не авторизован")
		return
	}
	userID, _ := userIDVal.(int64)

	user, err := h.storage.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	respondJSON(w, http.StatusOK, user.ToSafe())
}

// ListUsers — GET /api/users (admin only)
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.storage.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "ошибка получения пользователей")
		return
	}
	safe := make([]*SafeUser, 0, len(users))
	for _, u := range users {
		safe = append(safe, u.ToSafe())
	}
	respondJSON(w, http.StatusOK, safe)
}

// GetUser — GET /api/users/{id} (admin only)
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "неверный ID")
		return
	}
	user, err := h.storage.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	respondJSON(w, http.StatusOK, user.ToSafe())
}

// UpdateUser — PUT /api/users/{id} (admin only)
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "неверный ID")
		return
	}

	var input UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "ошибка разбора запроса")
		return
	}

	ctx := r.Context()
	user, err := h.storage.Update(ctx, id, &input)
	if err != nil {
		switch {
		case errors.Is(err, ErrPasswordTooShort):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrUserNotFound):
			respondError(w, http.StatusNotFound, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "ошибка обновления пользователя")
		}
		return
	}

	// Если пользователь деактивирован — отзываем все его токены
	if input.IsActive != nil && !*input.IsActive {
		_ = h.storage.RevokeAllUserTokens(ctx, id)
	}

	respondJSON(w, http.StatusOK, user.ToSafe())
}

// DeleteUser — DELETE /api/users/{id} (admin only)
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "неверный ID")
		return
	}

	ctx := r.Context()
	if err := h.storage.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "пользователь не найден")
		} else {
			respondError(w, http.StatusInternalServerError, "ошибка удаления")
		}
		return
	}

	_ = h.storage.RevokeAllUserTokens(ctx, id)
	respondJSON(w, http.StatusOK, map[string]string{"message": "пользователь удалён"})
}

// issueTokens — создаёт пару access+refresh токенов и сохраняет refresh в БД
func (h *Handler) issueTokens(ctx interface{ Done() <-chan struct{} }, user *User) (*TokenPair, error) {
	accessToken, err := h.jwt.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации access token: %w", err)
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    ExpiresIn(),
		User:         user.ToSafe(),
	}, nil
}

// Хелпер для wrapPublic — позволяет передавать handler без context
func (h *Handler) wrapPublic(fn http.HandlerFunc) http.Handler {
	return fn
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
