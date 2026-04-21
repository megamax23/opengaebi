package db_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/opengaebi/opengaebi/internal/db"
)

func TestSQLite_RegisterAndGet(t *testing.T) {
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	peer := db.Peer{
		ID:        "test-id",
		Workspace: "gowit",
		Name:      "ai-framework",
		Kind:      "session",
		Tags:      []string{"role:backend"},
		IP:        "127.0.0.1",
		Port:      8001,
	}

	if err := store.RegisterPeer(ctx, peer); err != nil {
		t.Fatalf("RegisterPeer failed: %v", err)
	}

	got, err := store.GetPeer(ctx, "gowit", "ai-framework")
	if err != nil {
		t.Fatalf("GetPeer failed: %v", err)
	}
	if got.ID != peer.ID {
		t.Errorf("expected ID=%s, got %s", peer.ID, got.ID)
	}
	if got.IP != peer.IP {
		t.Errorf("expected IP=%s, got %s", peer.IP, got.IP)
	}
}

func TestSQLite_ListPeers(t *testing.T) {
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	for i, name := range []string{"bot-a", "bot-b"} {
		store.RegisterPeer(ctx, db.Peer{
			ID: fmt.Sprintf("id-%d", i), Workspace: "gowit", Name: name, Kind: "agent",
		})
	}

	peers, err := store.ListPeers(ctx, "gowit")
	if err != nil {
		t.Fatalf("ListPeers failed: %v", err)
	}
	if len(peers) != 2 {
		t.Errorf("expected 2 peers, got %d", len(peers))
	}
}

func TestSQLite_DeletePeer(t *testing.T) {
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.RegisterPeer(ctx, db.Peer{ID: "del-id", Workspace: "gowit", Name: "temp", Kind: "agent"})

	if err := store.DeletePeer(ctx, "del-id"); err != nil {
		t.Fatalf("DeletePeer failed: %v", err)
	}

	got, err := store.GetPeer(ctx, "gowit", "temp")
	if err == nil || got != nil {
		t.Error("expected peer to be deleted")
	}
}
