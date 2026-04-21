package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/opengaebi/opengaebi/internal/db"
)

func TestPostgres_RegisterAndGet(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping postgres tests")
	}

	store, err := db.NewPostgres(context.Background(), url)
	if err != nil {
		t.Fatalf("failed to connect postgres: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	peer := db.Peer{
		ID: "pg-test-id", Workspace: "gowit", Name: "pg-bot",
		Kind: "agent", Tags: []string{"role:test"}, IP: "10.0.0.1", Port: 9000,
	}

	if err := store.RegisterPeer(ctx, peer); err != nil {
		t.Fatalf("RegisterPeer failed: %v", err)
	}
	defer store.DeletePeer(ctx, peer.ID)

	got, err := store.GetPeer(ctx, "gowit", "pg-bot")
	if err != nil {
		t.Fatalf("GetPeer failed: %v", err)
	}
	if got.ID != peer.ID {
		t.Errorf("expected ID=%s, got %s", peer.ID, got.ID)
	}
}
