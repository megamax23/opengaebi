package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/opengaebi/opengaebi/internal/a2a"
	"github.com/opengaebi/opengaebi/internal/db"
	"github.com/opengaebi/opengaebi/internal/mcp"
)

type Server struct {
	db      db.DB
	apiKey  string
	baseURL string
	mcpSrv  *mcp.Server
	a2aHndl *a2a.Handler
}

func New(store db.DB, apiKey string, baseURL string) *Server {
	return &Server{
		db:      store,
		apiKey:  apiKey,
		baseURL: baseURL,
		mcpSrv:  mcp.New(store),
		a2aHndl: a2a.New(store, baseURL),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// REST API (auth required)
	mux.Handle("POST /v1/agents", s.AuthMiddleware(http.HandlerFunc(s.registerAgent)))
	mux.Handle("GET /v1/agents", s.AuthMiddleware(http.HandlerFunc(s.listAgents)))
	mux.Handle("DELETE /v1/agents/{id}", s.AuthMiddleware(http.HandlerFunc(s.deleteAgent)))
	mux.Handle("POST /v1/messages", s.AuthMiddleware(http.HandlerFunc(s.sendMessage)))
	mux.Handle("GET /v1/messages", s.AuthMiddleware(http.HandlerFunc(s.pollMessages)))
	mux.Handle("DELETE /v1/messages/{id}", s.AuthMiddleware(http.HandlerFunc(s.deleteMessage)))
	mux.Handle("POST /v1/artifacts", s.AuthMiddleware(http.HandlerFunc(s.saveArtifact)))
	mux.Handle("GET /v1/artifacts/{id}", s.AuthMiddleware(http.HandlerFunc(s.getArtifact)))

	// MCP (no auth — clients use their own session context)
	mux.Handle("POST /mcp", s.mcpSrv)

	// A2A (no auth — open protocol)
	s.a2aHndl.Register(mux)

	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

const maxBodyBytes = 2 << 20 // 2MB 상한 — 파싱 전 메모리 소진 방지

func readJSON(r *http.Request, v any) error {
	// LimitReader: JSON 파싱 전 body 크기를 2MB로 제한, ResponseWriter 불필요
	return json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes)).Decode(v)
}

// Addr returns the listen address for the given port.
func (s *Server) Addr(port int) string {
	return fmt.Sprintf(":%d", port)
}
