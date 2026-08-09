package webserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"siem-server/actions"
	"siem-server/agent"
	"siem-server/alerts"
	wsevents "siem-server/internal/events"
	"siem-server/internal/storage/postgres"
	"siem-server/rules"
	"siem-server/users"

	"github.com/gorilla/mux"
)

//TODO: написать middleware для обработки паник в хендлерах

type WebServer struct {
	router        *mux.Router
	agentStorage  *agent.AgentStorage
	alertStorage  *alerts.AlertStorage
	actionStorage *actions.ActionStorage
	ruleStorage   *rules.RuleStorage
	logStorage    *postgres.LogStorage
	userStorage   *users.UserStorage
	jwtService    *users.JWTService
	userHandler   *users.Handler
	server        *http.Server
	eventBus      *wsevents.Bus
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
	eventBus *wsevents.Bus,
) *WebServer {
	ws := &WebServer{
		router:        mux.NewRouter(),
		agentStorage:  agentStorage,
		alertStorage:  alertStorage,
		actionStorage: actionStorage,
		ruleStorage:   ruleStorage,
		logStorage:    logStorage,
		userStorage:   userStorage,
		jwtService:    jwtService,
		userHandler:   userHandler,
		eventBus:      eventBus,
	}
	ws.setupRoutes()
	return ws
}

func (ws *WebServer) Start(addr string) error {
	ws.server = &http.Server{
		Addr:    addr,
		Handler: ws.router,
	}
	slog.Info("Web server starting on", "address", addr)
	if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server error: %w", err)
	}
	return nil
}

func (ws *WebServer) Shutdown(ctx context.Context) error {
	if ws.server == nil {
		return nil
	}
	if err := ws.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}
	return nil
}
