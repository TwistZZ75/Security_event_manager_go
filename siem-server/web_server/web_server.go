package webserver

import (
	"log"
	"net/http"
	"siem-server/actions"
	"siem-server/agent"
	"siem-server/alerts"
	"siem-server/internal/storage/postgres"
	"siem-server/rules"
	"siem-server/users"

	"github.com/gorilla/mux"
)

type WebServer struct {
	router        *mux.Router
	agentStorage  *agent.AgentStorage
	alertStorage  *alerts.AlertStorage
	actionStorage *actions.ActionStorage
	ruleStorage   *rules.RuleStorage
	LogStorage    *postgres.LogStorage
	userStorage   *users.UserStorage
	jwtService    *users.JWTService
	userHandler   *users.Handler
}

func NewWebServer(
	agentStorage *agent.AgentStorage,
	alertStorage *alerts.AlertStorage,
	actionStorage *actions.ActionStorage,
	ruleStorage *rules.RuleStorage,
	logStorage *postgres.LogStorage,
	userStorage *users.UserStorage,
	jwtService *users.JWTService,
	userHandler *users.Handler,
) *WebServer {
	ws := &WebServer{
		router:        mux.NewRouter(),
		agentStorage:  agentStorage,
		alertStorage:  alertStorage,
		actionStorage: actionStorage,
		ruleStorage:   ruleStorage,
		LogStorage:    logStorage,
		userStorage:   userStorage,
		jwtService:    jwtService,
		userHandler:   userHandler,
	}
	ws.setupRoutes()
	return ws
}

func (ws *WebServer) Start(addr string) error {
	log.Printf("Web server starting on %s", addr)
	return http.ListenAndServe(addr, ws.router)
}
