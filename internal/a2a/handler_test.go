package a2a_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opengaebi/opengaebi/internal/a2a"
	"github.com/opengaebi/opengaebi/internal/db"
)

func newA2AHandler(t *testing.T) *a2a.Handler {
	t.Helper()
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return a2a.New(store, "http://localhost:8080")
}

func TestA2A_AgentCard(t *testing.T) {
	h := newA2AHandler(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/.well-known/agent.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", rec.Code, rec.Body.String())
	}
	var card map[string]any
	json.NewDecoder(rec.Body).Decode(&card)
	if card["name"] == nil {
		t.Errorf("expected name in agent card: %+v", card)
	}
	if card["url"] == nil {
		t.Errorf("expected url in agent card: %+v", card)
	}
}

func TestA2A_TaskSend(t *testing.T) {
	h := newA2AHandler(t)
	mux := http.NewServeMux()
	h.Register(mux)

	body := map[string]any{
		"id":        "task-001",
		"message":   map[string]any{"role": "user", "content": "hello"},
		"workspace": "a2a-ws",
		"from":      "session-a",
		"to":        "bot-b",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/tasks/send", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["id"] == nil {
		t.Errorf("expected id in response: %+v", resp)
	}
	if resp["status"] == nil {
		t.Errorf("expected status in response: %+v", resp)
	}
}

func TestA2A_TaskSend_MissingFields(t *testing.T) {
	h := newA2AHandler(t)
	mux := http.NewServeMux()
	h.Register(mux)

	body := map[string]any{"id": "task-002"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/tasks/send", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
