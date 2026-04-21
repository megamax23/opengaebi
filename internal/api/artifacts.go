package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/opengaebi/opengaebi/internal/db"
)

const maxArtifactBytes = 1 << 20 // 1MB

type saveArtifactReq struct {
	Workspace string `json:"workspace"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Content   string `json:"content"`
}

func (s *Server) saveArtifact(w http.ResponseWriter, r *http.Request) {
	var req saveArtifactReq
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Workspace == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace and name are required"})
		return
	}
	if len(req.Content) > maxArtifactBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "content exceeds 1MB limit"})
		return
	}
	kind := req.Kind
	if kind == "" {
		kind = "text"
	}
	art := db.Artifact{
		ID:        uuid.New().String(),
		Workspace: req.Workspace,
		Name:      req.Name,
		Kind:      kind,
		Content:   []byte(req.Content),
	}
	if err := s.db.SaveArtifact(r.Context(), art); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": art.ID})
}

func (s *Server) getArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	art, err := s.db.GetArtifact(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        art.ID,
		"workspace": art.Workspace,
		"name":      art.Name,
		"kind":      art.Kind,
		"content":   string(art.Content),
	})
}
