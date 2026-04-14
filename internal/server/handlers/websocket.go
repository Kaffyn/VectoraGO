package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/Kaffyn/Vectora/internal/core/manager"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, add domain validation
	},
}

type WSHandler struct {
	tenantMgr *manager.TenantManager
}

func NewWSHandler(tm *manager.TenantManager) *WSHandler {
	return &WSHandler{tenantMgr: tm}
}

type WSRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type WSResponse struct {
	ID     string      `json:"id,omitempty"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func (h *WSHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to upgrade websocket")
		return
	}
	defer conn.Close()

	log.Info().Msg("New websocket connection established")

	var mu sync.Mutex
	write := func(resp WSResponse) {
		mu.Lock()
		defer mu.Unlock()
		if err := conn.WriteJSON(resp); err != nil {
			log.Error().Err(err).Msg("failed to write ws message")
		}
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Info().Msg("Websocket connection closed")
			break
		}

		var req WSRequest
		if err := json.Unmarshal(message, &req); err != nil {
			write(WSResponse{Error: "Invalid JSON-RPC format"})
			continue
		}

		go h.processRequest(context.Background(), req, write)
	}
}

func (h *WSHandler) processRequest(ctx context.Context, req WSRequest, write func(WSResponse)) {
	switch req.Method {
	case "ping":
		write(WSResponse{ID: req.ID, Result: "pong"})
	
	case "workspace.query":
		var params struct {
			Query         string `json:"query"`
			WorkspaceID   string `json:"workspace_id"`
			Model         string `json:"model"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			write(WSResponse{ID: req.ID, Error: "Invalid params"})
			return
		}

		// Resolve tenant
		tenant, err := h.tenantMgr.GetOrCreateTenant(params.WorkspaceID, ".", "default")
		if err != nil {
			write(WSResponse{ID: req.ID, Error: fmt.Sprintf("failed to resolve tenant: %v", err)})
			return
		}

		// Process query
		respChan, err := tenant.Engine.StreamQuery(ctx, params.Query, params.WorkspaceID, params.Model, "agent", "en")
		if err != nil {
			write(WSResponse{ID: req.ID, Error: err.Error()})
			return
		}

		var fullAnswer string
		for chunk := range respChan {
			if chunk.Token != "" {
				fullAnswer += chunk.Token
				// Notification for streaming
				write(WSResponse{
					ID: req.ID,
					Result: map[string]string{
						"token": chunk.Token,
						"type":  "token",
					},
				})
			}
		}

		write(WSResponse{ID: req.ID, Result: map[string]string{
			"answer": fullAnswer,
			"type":   "complete",
		}})

	default:
		write(WSResponse{ID: req.ID, Error: "Method not found"})
	}
}
