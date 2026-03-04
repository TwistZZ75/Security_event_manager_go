package webserver

import "net/http"

// setupRoutes настраивает маршруты
func (ws *WebServer) setupRoutes() {
	// CORS middleware
	ws.router.Use(corsMiddleware)

	// API routes
	api := ws.router.PathPrefix("/api").Subrouter()

	// Auth
	api.HandleFunc("/auth/login", ws.handleLogin).Methods("POST", "OPTIONS")

	// Agents
	api.HandleFunc("/agents", ws.handleGetAgents).Methods("GET", "OPTIONS")
	api.HandleFunc("/agents/{id}", ws.handleGetAgent).Methods("GET", "OPTIONS")

	// Alerts
	api.HandleFunc("/alerts", ws.handleGetAlerts).Methods("GET", "OPTIONS")
	api.HandleFunc("/alerts/{id}", ws.handleGetAlert).Methods("GET", "OPTIONS")

	// Actions
	api.HandleFunc("/actions", ws.handleGetActions).Methods("GET", "OPTIONS")
	api.HandleFunc("/actions/{id}", ws.handleGetAction).Methods("GET", "OPTIONS")

	// Rules
	api.HandleFunc("/rules", ws.handleGetRules).Methods("GET", "OPTIONS")
	api.HandleFunc("/rules", ws.handleCreateRule).Methods("POST", "OPTIONS")
	api.HandleFunc("/rules/{id}", ws.handleGetRule).Methods("GET", "OPTIONS")
	api.HandleFunc("/rules/{id}", ws.handleUpdateRule).Methods("PUT", "OPTIONS")
	api.HandleFunc("/rules/{id}", ws.handleDeleteRule).Methods("DELETE", "OPTIONS")

	// Static files (serve the web interface)
	ws.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web_server/static")))
}
