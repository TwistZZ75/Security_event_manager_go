package webserver

import (
	"log"
	"net/http"
	"siem-server/actions"
	"siem-server/agent"
	"siem-server/alerts"
	"siem-server/internal/storage/postgres"
	"siem-server/rules"

	"github.com/gorilla/mux"
)

// WebServer обслуживает HTTP API и веб-интерфейс
type WebServer struct {
	router        *mux.Router
	agentStorage  *agent.AgentStorage
	alertStorage  *alerts.AlertStorage
	actionStorage *actions.ActionStorage
	ruleStorage   *rules.RuleStorage
	LogStorage    *postgres.LogStorage
}

// NewWebServer создает новый веб-сервер
func NewWebServer(
	agentStorage *agent.AgentStorage,
	alertStorage *alerts.AlertStorage,
	actionStorage *actions.ActionStorage,
	ruleStorage *rules.RuleStorage,
	LogStorage *postgres.LogStorage,
) *WebServer {
	ws := &WebServer{
		router:        mux.NewRouter(),
		agentStorage:  agentStorage,
		alertStorage:  alertStorage,
		actionStorage: actionStorage,
		ruleStorage:   ruleStorage,
		LogStorage:    LogStorage,
	}

	ws.setupRoutes()
	return ws
}

// Start запускает веб-сервер
func (ws *WebServer) Start(addr string) error {
	log.Printf("Web server starting on %s", addr)
	return http.ListenAndServe(addr, ws.router)
}
