package webserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"siem-server/rules"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

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

func (ws *WebServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// TODO: Implement proper authentication
	if req.Username == "admin" && req.Password == "admin" {
		respondWithJSON(w, http.StatusOK, LoginResponse{
			Success: true,
			Token:   "dummy-token-" + time.Now().Format("20060102150405"),
			Message: "Login successful",
		})
	} else {
		respondWithJSON(w, http.StatusUnauthorized, LoginResponse{
			Success: false,
			Message: "Invalid credentials",
		})
	}
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
	alertIDStr := vars["id"]

	alertID, err := strconv.ParseInt(alertIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid alert ID")
		return
	}

	var body struct {
		Status    string `json:"status"`
		Notes     string `json:"notes"`
		UpdatedBy string `json:"updated_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	//если фронт не передал пользователя
	if body.UpdatedBy == "" {
		body.UpdatedBy = "web"
	}

	ctx := r.Context()
	if err := ws.alertStorage.UpdateAlert(ctx, alertID, body.Status, "web", body.Notes); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update alert")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":         alertID,
		"status":     body.Status,
		"updated_by": body.UpdatedBy,
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

	logs, err := ws.LogStorage.GetRecent(ctx, 500)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch events")
		return
	}

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
		})
	}

	respondWithJSON(w, http.StatusOK, events)
}
