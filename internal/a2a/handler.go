package a2a

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/opengaebi/opengaebi/internal/db"
)

type Handler struct {
	db      db.DB
	baseURL string
}

func New(store db.DB, baseURL string) *Handler {
	return &Handler{db: store, baseURL: baseURL}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/agent.json", h.agentCard)
	mux.HandleFunc("POST /tasks/send", h.taskSend)
}

func (h *Handler) agentCard(w http.ResponseWriter, r *http.Request) {
	card := map[string]any{
		"name":        "opengaebi-bridge",
		"description": "AI agent message broker and registry bridge",
		"url":         h.baseURL,
		"version":     "0.1.0",
		"capabilities": map[string]any{
			"streaming":              false,
			"pushNotifications":      false,
			"stateTransitionHistory": false,
		},
	}
	writeJSON(w, http.StatusOK, card)
}

type taskSendReq struct {
	ID        string         `json:"id"`
	Message   map[string]any `json:"message"`
	Workspace string         `json:"workspace"`
	From      string         `json:"from"`
	To        string         `json:"to"`
}

func (h *Handler) taskSend(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read error"})
		return
	}
	var req taskSendReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Workspace == "" || req.To == "" || req.Message == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace, to, and message are required"})
		return
	}

	payload, _ := json.Marshal(req.Message)
	msg := db.Message{
		ID:        uuid.New().String(),
		FromPeer:  req.From,
		ToPeer:    req.To,
		Workspace: req.Workspace,
		Payload:   string(payload),
	}
	if err := h.db.SendMessage(r.Context(), msg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":     msg.ID,
		"status": map[string]any{"state": "submitted"},
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
