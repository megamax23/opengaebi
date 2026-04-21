package api

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/opengaebi/opengaebi/internal/db"
)

type sendMessageReq struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Workspace string `json:"workspace"`
	Payload   string `json:"payload"`
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	var req sendMessageReq
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Workspace == "" || req.To == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace and to are required"})
		return
	}
	if len(req.Payload) > 64*1024 {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "payload exceeds 64KB limit"})
		return
	}
	msg := db.Message{
		ID:        uuid.New().String(),
		FromPeer:  req.From,
		ToPeer:    req.To,
		Workspace: req.Workspace,
		Payload:   req.Payload,
	}
	if err := s.db.SendMessage(r.Context(), msg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": msg.ID})
}

func (s *Server) pollMessages(w http.ResponseWriter, r *http.Request) {
	workspace := r.URL.Query().Get("workspace")
	to := r.URL.Query().Get("to")
	if workspace == "" || to == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace and to are required"})
		return
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	msgs, err := s.db.PollMessages(r.Context(), workspace, to, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if msgs == nil {
		msgs = []db.Message{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
}

func (s *Server) deleteMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.DeleteMessage(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
