package registry_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opengaebi/opengaebi/internal/registry"
)

func TestClient_Push_SkipsWhenNoURL(t *testing.T) {
	c := registry.NewClient("", "key")
	err := c.PushPeer(context.Background(), registry.PeerPayload{
		ID: "p1", Name: "bot", Kind: "agent",
	})
	if err != nil {
		t.Errorf("expected no error when URL empty, got: %v", err)
	}
}

func TestClient_Push_CallsRegistry(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "p1"})
	}))
	defer srv.Close()

	c := registry.NewClient(srv.URL, "test-key")
	err := c.PushPeer(context.Background(), registry.PeerPayload{
		ID: "p1", Name: "bot", Kind: "agent",
		Tags: []string{"role:worker"}, IP: "10.0.0.1", Port: 9000,
	})
	if err != nil {
		t.Fatalf("PushPeer: %v", err)
	}
	if received["name"] != "bot" {
		t.Errorf("unexpected payload: %+v", received)
	}
}

func TestClient_DeletePeer(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			called = true
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := registry.NewClient(srv.URL, "key")
	c.DeletePeer(context.Background(), "peer-id")
	if !called {
		t.Error("expected DELETE to be called")
	}
}
