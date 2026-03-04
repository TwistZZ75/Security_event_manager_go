package webserver

import (
	"encoding/json"
	"net/http"
	"siem-server/rules"
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
