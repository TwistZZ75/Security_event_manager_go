package webserver

import (
	"net/http"
	"siem-server/users"
)

func (ws *WebServer) setupRoutes() {
	// CORS middleware применяется глобально
	ws.router.Use(corsMiddleware)

	api := ws.router.PathPrefix("/api").Subrouter()

	// ── Middleware factories ────────────────────────────────────────────────
	authRequired := AuthMiddleware(ws.jwtService)
	adminOnly := RequireRole(users.RoleAdmin)
	analystUp := RequireRole(users.RoleAdmin, users.RoleAnalyst)

	// ── Аутентификация (публичные) ──────────────────────────────────────────
	// Регистрация, вход, обновление токена — без авторизации
	ws.userHandler.RegisterRoutes(api, authRequired, adminOnly)
	//   Этот метод регистрирует:
	//   POST /api/auth/register  — публичный
	//   POST /api/auth/login     — публичный
	//   POST /api/auth/refresh   — публичный
	//   POST /api/auth/logout    — требует токен
	//   GET  /api/auth/me        — требует токен
	//   GET  /api/users          — только admin
	//   GET  /api/users/{id}     — только admin
	//   PUT  /api/users/{id}     — только admin
	//   DELETE /api/users/{id}   — только admin

	// ── Защищённые маршруты ─────────────────────────────────────────────────
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authRequired)

	// Агенты (только чтение для всех ролей)
	protected.HandleFunc("/agents", ws.handleGetAgents).Methods("GET", "OPTIONS")
	protected.HandleFunc("/agents/{id}", ws.handleGetAgent).Methods("GET", "OPTIONS")

	// Алерты
	protected.HandleFunc("/alerts", ws.handleGetAlerts).Methods("GET", "OPTIONS")
	protected.HandleFunc("/alerts/{id}", ws.handleGetAlert).Methods("GET", "OPTIONS")
	// Изменение статуса алерта — analyst и выше
	protected.Handle("/alerts/{id}/status",
		RequireRole(users.RoleAdmin, users.RoleAnalyst)(
			http.HandlerFunc(ws.handleUpdateAlertStatus),
		),
	).Methods("PATCH", "OPTIONS")

	// Действия (только чтение)
	protected.HandleFunc("/actions", ws.handleGetActions).Methods("GET", "OPTIONS")
	protected.HandleFunc("/actions/{id}", ws.handleGetAction).Methods("GET", "OPTIONS")

	// Правила — чтение для всех, запись для analyst+
	protected.HandleFunc("/rules", ws.handleGetRules).Methods("GET", "OPTIONS")
	protected.HandleFunc("/rules/{id}", ws.handleGetRule).Methods("GET", "OPTIONS")

	rulesWrite := protected.PathPrefix("").Subrouter()
	rulesWrite.Use(analystUp)
	rulesWrite.HandleFunc("/rules", ws.handleCreateRule).Methods("POST", "OPTIONS")
	rulesWrite.HandleFunc("/rules/{id}", ws.handleUpdateRule).Methods("PUT", "OPTIONS")
	rulesWrite.HandleFunc("/rules/{id}/enabled", ws.handleSetRuleEnabled).Methods("PATCH", "OPTIONS")

	// Удаление правил — только admin
	protected.Handle("/rules/{id}",
		RequireRole(users.RoleAdmin)(
			http.HandlerFunc(ws.handleDeleteRule),
		),
	).Methods("DELETE", "OPTIONS")

	// События
	protected.HandleFunc("/events", ws.handleGetEvents).Methods("GET", "OPTIONS")

	// Статические файлы
	ws.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web_server/static")))
}
