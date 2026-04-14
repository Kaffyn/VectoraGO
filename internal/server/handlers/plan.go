package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Kaffyn/Vectora/internal/core/manager"
)

// PlanRequest representa uma requisição para gerar um plano
type PlanRequest struct {
	Description string            `json:"description"`
	WorkspaceID string            `json:"workspace_id"`
	Context     map[string]string `json:"context,omitempty"`
	Scope       string            `json:"scope,omitempty"` // minimal, standard, comprehensive
	MaxSteps    int               `json:"max_steps,omitempty"`
}

// PlanStep representa um passo dentro de um plano
type PlanStep struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Status        string        `json:"status"` // pending, in_progress, completed
	Dependencies  []string      `json:"dependencies,omitempty"`
	EstimatedTime int           `json:"estimated_time_minutes"`
	RiskLevel     string        `json:"risk_level"` // low, medium, high
	Notes         string        `json:"notes,omitempty"`
}

// PlanResponse representa a resposta com um plano gerado
type PlanResponse struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Steps         []PlanStep  `json:"steps"`
	EstimatedTime int         `json:"estimated_time_minutes"`
	Complexity    string      `json:"complexity"` // low, medium, high
	GeneratedAt   time.Time   `json:"generated_at"`
}

// Validate valida a requisição de plano
func (pr *PlanRequest) Validate() error {
	if pr.Description == "" {
		return fmt.Errorf("description is required")
	}

	if pr.WorkspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}

	if pr.Scope != "" && pr.Scope != "minimal" && pr.Scope != "standard" && pr.Scope != "comprehensive" {
		return fmt.Errorf("scope must be minimal, standard, or comprehensive")
	}

	if pr.MaxSteps < 0 {
		return fmt.Errorf("max_steps must be >= 0")
	}

	return nil
}

// PlanHandler processa requisições de planejamento
type PlanHandler struct {
	tenantMgr *manager.TenantManager
}

// NewPlanHandler cria um novo plan handler
func NewPlanHandler(tm *manager.TenantManager) *PlanHandler {
	return &PlanHandler{
		tenantMgr: tm,
	}
}

// HandleCreatePlan processa POST /api/plan
func (h *PlanHandler) HandleCreatePlan(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)
	_ = GetUserIDFromContext(r)

	var req PlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err), requestID)
		return
	}

	if err := req.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error(), requestID)
		return
	}

	// Obtém o tenant para o workspace
	tenant, err := h.tenantMgr.GetTenant(req.WorkspaceID)
	if err != nil {
		WriteError(w, http.StatusNotFound, fmt.Sprintf("workspace %s not found", req.WorkspaceID), requestID)
		return
	}

	// Chamar engine para gerar plano (usando modo planning)
	answer, _, err := tenant.Engine.Query(r.Context(), req.Description, req.WorkspaceID, "", "planning", "standard")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate plan: %v", err), requestID)
		return
	}

	// Como a resposta do engine é uma string (markdown ou texto), precisamos converter
	// para o formato estruturado do PlanResponse se possível, ou apenas retornar o texto.
	// Por agora, retornamos o texto como descrição e steps vazios (ou parseados).
	resp := PlanResponse{
		ID:          "plan_" + requestID[:8],
		Title:       "Generated Plan",
		Description: answer,
		Complexity:  "medium",
		GeneratedAt: time.Now(),
		Steps:       []PlanStep{}, // TODO: Implementar parser de markdown para steps
	}

	WriteJSON(w, http.StatusCreated, resp, requestID)
}

// HandleGetPlan retorna um plano específico
func (h *PlanHandler) HandleGetPlan(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	planID := r.URL.Query().Get("plan_id")
	if planID == "" {
		WriteError(w, http.StatusBadRequest, "plan_id is required", requestID)
		return
	}

	// TODO: Recuperar do plan store
	mockPlan := PlanResponse{
		ID:    planID,
		Title: "Retrieved Plan",
		Description: "This is a mock plan retrieved from storage",
		Steps: []PlanStep{
			{
				ID:            "step_1",
				Title:         "Initial Setup",
				Description:   "Setup infrastructure",
				Status:        "completed",
				EstimatedTime: 45,
				RiskLevel:     "low",
			},
		},
		EstimatedTime: 45,
		Complexity:    "low",
		GeneratedAt:   time.Now().Add(-24 * time.Hour),
	}

	WriteJSON(w, http.StatusOK, mockPlan, requestID)
}

// HandleUpdatePlanStep atualiza o status de um step
func (h *PlanHandler) HandleUpdatePlanStep(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	var updateReq struct {
		PlanID string `json:"plan_id"`
		StepID string `json:"step_id"`
		Status string `json:"status"` // pending, in_progress, completed
		Notes  string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err), requestID)
		return
	}

	if updateReq.PlanID == "" || updateReq.StepID == "" || updateReq.Status == "" {
		WriteError(w, http.StatusBadRequest, "plan_id, step_id, and status are required", requestID)
		return
	}

	// TODO: Atualizar no plan store
	mockResp := map[string]interface{}{
		"plan_id":     updateReq.PlanID,
		"step_id":     updateReq.StepID,
		"status":      updateReq.Status,
		"updated_at":  time.Now(),
		"request_id":  requestID,
	}

	WriteJSON(w, http.StatusOK, mockResp, requestID)
}

// HandleListPlans lista planos do usuário
func (h *PlanHandler) HandleListPlans(w http.ResponseWriter, r *http.Request) {
	requestID := GetRequestID(r)

	// TODO: Recuperar planos do usuário do plan store
	mockPlans := []map[string]interface{}{
		{
			"id":          "plan_001",
			"title":       "Phase 4 Implementation",
			"description": "Cloud-native API development",
			"status":      "in_progress",
			"created_at":  time.Now().Add(-7 * 24 * time.Hour),
		},
		{
			"id":          "plan_002",
			"title":       "Database Optimization",
			"description": "Query performance tuning",
			"status":      "completed",
			"created_at":  time.Now().Add(-30 * 24 * time.Hour),
		},
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"plans": mockPlans,
		"count": len(mockPlans),
	}, requestID)
}
