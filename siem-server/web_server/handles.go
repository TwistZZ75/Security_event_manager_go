package webserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	wsevents "siem-server/internal/events"
	logstructure "siem-server/internal/logsstructure"
	"siem-server/rules"
	"siem-server/users"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/gorilla/mux"
)

// ============================================================================
// HANDLERS - WEBSOCKET
// ============================================================================

type authMsg struct {
	Token string `json:"token"`
}

type subscribeMsg struct {
	Types  []string `json:"type"`
	Action string   `json:"action"`
}

const maxMessageSize = 4096

// ============================================================================
// HANDLERS - AUTH
// ============================================================================

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
}

// ============================================================================
// HANDLERS - AGENTS
// ============================================================================

func (ws *WebServer) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	agents, err := ws.agentStorage.ListAgents(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch agents")
		return
	}

	respondWithJSON(w, http.StatusOK, agents)
}

func (ws *WebServer) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	ctx := r.Context()
	agent, err := ws.agentStorage.GetAgent(ctx, agentID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}

	respondWithJSON(w, http.StatusOK, agent)
}

// ============================================================================
// HANDLERS - ALERTS
// ============================================================================

func (ws *WebServer) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	alerts, err := ws.alertStorage.GetRecentAlerts(ctx, 100)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch alerts")
		return
	}

	respondWithJSON(w, http.StatusOK, alerts)
}

func (ws *WebServer) handleGetAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	ctx := r.Context()
	alert, err := ws.alertStorage.GetAlert(ctx, alertID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Alert not found")
		return
	}

	respondWithJSON(w, http.StatusOK, alert)
}

func (ws *WebServer) handleUpdateAlertStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid alert ID")
		return
	}

	var body struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// НОВОЕ: берём имя пользователя из контекста (кладётся middleware)
	updatedBy, _ := getUsername(r.Context())
	if updatedBy == "" {
		updatedBy = "web"
	}

	ctx := r.Context()
	if err := ws.alertStorage.UpdateAlert(ctx, alertID, body.Status, updatedBy, body.Notes); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update alert")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":         alertID,
		"status":     body.Status,
		"updated_by": updatedBy,
	})
}

// ============================================================================
// HANDLERS - ACTIONS
// ============================================================================

func (ws *WebServer) handleGetActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actions, err := ws.actionStorage.GetRecentActions(ctx, 100)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch actions")
		return
	}

	respondWithJSON(w, http.StatusOK, actions)
}

func (ws *WebServer) handleGetAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	actionID := vars["id"]

	ctx := r.Context()
	action, err := ws.actionStorage.GetActionLog(ctx, actionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Action not found")
		return
	}

	respondWithJSON(w, http.StatusOK, action)
}

// ============================================================================
// HANDLERS - RULES
// ============================================================================

func (ws *WebServer) handleGetRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rules, err := ws.ruleStorage.GetAllRules(ctx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch rules")
		return
	}

	respondWithJSON(w, http.StatusOK, rules)
}

func (ws *WebServer) handleGetRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	ctx := r.Context()
	rule, err := ws.ruleStorage.GetRule(ctx, ruleID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Rule not found")
		return
	}

	respondWithJSON(w, http.StatusOK, rule)
}

func (ws *WebServer) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var rule rules.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	var aggregationJSON []byte

	conditionsJSON, _ := json.Marshal(rule.Conditions)
	if rule.Aggregation != nil {
		aggregationJSON, _ = json.Marshal(rule.Aggregation)
	}

	// Генерируем ID если не задан
	if rule.ID == "" {
		var raw string
		if rule.Aggregation != nil {
			raw = rule.Name + rule.Severity + string(conditionsJSON) + string(aggregationJSON)
		} else {
			raw = rule.Name + rule.Severity + string(conditionsJSON)
		}
		hash := sha256.Sum256([]byte(raw))
		rule.ID = hex.EncodeToString(hash[:])
	}

	// Устанавливаем время создания если не задано
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	if rule.CreatedBy == "" {
		rule.CreatedBy = "web"
	}

	ctx := r.Context()
	if err := ws.ruleStorage.SaveRule(ctx, &rule); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create rule")
		return
	}

	respondWithJSON(w, http.StatusCreated, rule)
}

func (ws *WebServer) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	var rule rules.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	rule.ID = ruleID

	ctx := r.Context()
	if err := ws.ruleStorage.SaveRule(ctx, &rule); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update rule")
		return
	}

	respondWithJSON(w, http.StatusOK, rule)
}

func (ws *WebServer) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	ctx := r.Context()
	if err := ws.ruleStorage.RemoveRule(ctx, ruleID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Rule deleted successfully"})
}

func (ws *WebServer) handleSetRuleEnabled(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	if err := ws.ruleStorage.SetRuleEnabled(ctx, ruleID, body.Enabled); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update rule")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":      ruleID,
		"enabled": body.Enabled,
	})
}

// ============================================================================
// HANDLERS - WEBSOCKET
// ============================================================================

func (ws *WebServer) handlerWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{
			"localhost:3000",
			"127.0.0.1:3000",
		},
	}) // принимаем и апгрейдим наше соединение
	if err != nil {
		slog.Error("Cannot accept websocket connect", "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "connection closed")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	conn.SetReadLimit(maxMessageSize) // ограничение размера входящих сообщений для предотвращения атак большими сообщениями

	// не проходит аунтетификация через middleware потому что там мы берём токен из заголовка request, а здесь из
	// массива байт пересылаемого в первом сообщении после открытого websocket соединения
	authCtx, authCancel := context.WithTimeout(ctx, 5*time.Second)

	claims, err := authCheck(authCtx, conn, ws.jwtService)
	authCancel()
	if err != nil {
		slog.Warn("websocket authentication failed", "error", err)
		conn.Close(websocket.StatusPolicyViolation, "authentication failed")
		return
	}

	slog.Info("websocket authenticated",
		"user_id", claims.UserID,
		"username", claims.Username,
		"role", claims.Role,
	)

	sub := ws.eventBus.Subscribe(func(e wsevents.Event) bool {
		switch e.Type {
		case wsevents.EventCreated,
			wsevents.AlertCreated,
			wsevents.AlertUpdated:
			return true
		default:
			return false
		}
	})

	if sub == nil {
		_ = conn.Close(websocket.StatusInternalError, "event bus is shuting down")
		return
	}

	defer ws.eventBus.Unsubscribe(sub.ID)

	// writter
	writeDone := make(chan struct{})

	go func() {
		defer close(writeDone)

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-sub.Channel:
				if !ok {
					return
				}

				if conn.Write(ctx, websocket.MessageText, mustJSON(event)); err != nil {
					slog.Debug("websocket write failed", "error", err)
					cancel()
					return
				}

			}
		}
	}()

	//reader
	for {
		//TODO: должны принимать сообщения от фронта
		// сообщения типа subscribe, unsubscribe, filters
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	<-writeDone
	slog.Debug("websocket connection closed", "user_id", claims.UserID)
}

func authCheck(authCtx context.Context, conn *websocket.Conn, jwtService *users.JWTService) (*users.JWTClaims, error) {
	_, msg, err := conn.Read(authCtx)
	if err != nil {
		slog.Error("Cannot read websocket open message", "error", err)
		conn.Close(websocket.StatusPolicyViolation, "auth timeout")
		return nil, fmt.Errorf("auth timeout, %w", err)
	}

	var authMsg authMsg

	if err := json.Unmarshal(msg, &authMsg); err != nil {
		slog.Error("Unmarshal ws auth token msg", "error", err)
		conn.Close(websocket.StatusPolicyViolation, "invalid auth message")
		return nil, fmt.Errorf("invalid auth message, %w", err)
	}

	if authMsg.Token == "" {
		return nil, fmt.Errorf("empty token")
	}

	claims, err := jwtService.ValidateAccessToken(authMsg.Token)
	if err != nil {
		slog.Error("Invalid token", "error", err)
		conn.Close(websocket.StatusPolicyViolation, "invalid token")
		return nil, fmt.Errorf("invalid token, %w", err)
	}

	return claims, nil
}

func mustJSON(event wsevents.Event) []byte {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("cannot marshal websocket event", "error", err)
		return []byte(`{"type":"error"}`)
	}
	return data
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// ============================================================================
// HANDLERS - EVENTS (normalized_events)
// ============================================================================

func (ws *WebServer) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Читаем ?limit=N
	limit := 500
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if n, err := strconv.Atoi(lStr); err == nil && n > 0 {
			if n > 10000 {
				n = 10000 // защита от слишком большого запроса
			}
			limit = n
		}
	}

	logs, err := ws.logStorage.GetRecent(ctx, limit)
	if err != nil {
		log.Printf("handleGetEvents: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch events")
		return
	}
	if logs == nil {
		logs = []*logstructure.NormalizedLog{}
	}

	// Маппинг в Event для фронтенда
	type Event struct {
		ID               string `json:"id"`
		PCName           string `json:"pc_name"`
		Username         string `json:"username"`
		EventDescription string `json:"event_description"`
		EventCategory    string `json:"event_category"`
		ProcessName      string `json:"process_name"`
		Severity         string `json:"severity"`
		Timestamp        string `json:"timestamp"`
		OS               string `json:"os"`
		Source           string `json:"source"`
		RawLog           string `json:"raw_log"`
	}

	events := make([]Event, 0, len(logs))
	for _, l := range logs {
		events = append(events, Event{
			ID:               l.ID,
			PCName:           l.PC_name,
			Username:         l.Username,
			EventDescription: l.Event_description,
			EventCategory:    l.Event_category,
			ProcessName:      l.Process_name,
			Severity:         l.Severity,
			Timestamp:        l.Timestamp.Format(time.RFC3339),
			OS:               l.OS,
			Source:           l.Source,
			RawLog:           l.Raw_log,
		})
	}

	respondWithJSON(w, http.StatusOK, events)
}
