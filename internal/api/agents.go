package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/opengaebi/opengaebi/internal/db"
)

type registerAgentReq struct {
	Workspace string   `json:"workspace"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Tags      []string `json:"tags"`
	IP        string   `json:"ip"`
	Port      int      `json:"port"`
}

func (s *Server) registerAgent(w http.ResponseWriter, r *http.Request) {
	var req registerAgentReq
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Workspace == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace and name are required"})
		return
	}
	kind := req.Kind
	if kind == "" {
		kind = "session"
	}
	peer := db.Peer{
		ID:        uuid.New().String(),
		Workspace: req.Workspace,
		Name:      req.Name,
		Kind:      kind,
		Tags:      req.Tags,
		IP:        req.IP,
		Port:      req.Port,
		LastSeen:  time.Now(),
	}
	if err := s.db.RegisterPeer(r.Context(), peer); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": peer.ID})
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	workspace := r.URL.Query().Get("workspace")
	if workspace == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace is required"})
		return
	}
	peers, err := s.db.ListPeers(r.Context(), workspace)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if peers == nil {
		peers = []db.Peer{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"peers": peers})
}

func (s *Server) deleteAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.DeletePeer(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
