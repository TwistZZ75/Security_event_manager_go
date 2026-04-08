package webserver

import (
	"context"
	"net/http"
	"siem-server/users"
	"strings"
)

// Ключи контекста
type contextKey int

const (
	ContextUserID   contextKey = iota
	ContextUsername contextKey = iota
	ContextUserRole contextKey = iota
)

// AuthMiddleware — проверяет JWT в заголовке Authorization: Bearer <token>
func AuthMiddleware(jwtSvc *users.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// preflight — пропускаем
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w, "токен не предоставлен")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				respondUnauthorized(w, "неверный формат токена")
				return
			}

			claims, err := jwtSvc.ValidateAccessToken(parts[1])
			if err != nil {
				switch err {
				case users.ErrTokenExpired:
					respondUnauthorized(w, "токен истёк")
				default:
					respondUnauthorized(w, "недействительный токен")
				}
				return
			}

			// Кладём данные пользователя в контекст
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextUsername, claims.Username)
			ctx = context.WithValue(ctx, ContextUserRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole — middleware для проверки роли
func RequireRole(roles ...users.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(ContextUserRole).(users.Role)
			if !ok {
				respondForbidden(w, "роль не определена")
				return
			}

			for _, allowed := range roles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			respondForbidden(w, "недостаточно прав")
		})
	}
}

// RequirePermission — middleware для проверки конкретного разрешения
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(ContextUserRole).(users.Role)
			if !ok || !role.HasPermission(permission) {
				respondForbidden(w, "нет разрешения: "+permission)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID — хелпер для извлечения ID пользователя из контекста
func GetUserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ContextUserID).(int64)
	return id, ok
}

// GetUsername — хелпер для извлечения имени пользователя из контекста
func GetUsername(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(ContextUsername).(string)
	return name, ok
}

// GetUserRole — хелпер для извлечения роли из контекста
func GetUserRole(ctx context.Context) (users.Role, bool) {
	role, ok := ctx.Value(ContextUserRole).(users.Role)
	return role, ok
}

func respondUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + msg + `"}`))
}

func respondForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":"` + msg + `"}`))
}
