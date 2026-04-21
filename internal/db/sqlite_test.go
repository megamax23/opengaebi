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

func TestSQLite_SendAndPollMessages(t *testing.T) {
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	msg := db.Message{
		ID:        "msg-1",
		FromPeer:  "agent-a",
		ToPeer:    "agent-b",
		Workspace: "gowit",
		Payload:   `{"text":"hello"}`,
	}

	if err := store.SendMessage(ctx, msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	msgs, err := store.PollMessages(ctx, "gowit", "agent-b", 10)
	if err != nil {
		t.Fatalf("PollMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Payload != msg.Payload {
		t.Errorf("payload mismatch: got %s", msgs[0].Payload)
	}
}

func TestSQLite_DeleteMessage(t *testing.T) {
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.SendMessage(ctx, db.Message{ID: "del-msg", FromPeer: "a", ToPeer: "b", Workspace: "ws", Payload: "x"})

	if err := store.DeleteMessage(ctx, "del-msg"); err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}

	msgs, _ := store.PollMessages(ctx, "ws", "b", 10)
	if len(msgs) != 0 {
		t.Error("expected 0 messages after delete")
	}
}

func TestSQLite_SaveAndGetArtifact(t *testing.T) {
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	art := db.Artifact{
		ID:        "art-1",
		Workspace: "gowit",
		Name:      "schema.json",
		Kind:      "code",
		Content:   []byte(`{"version":1}`),
	}

	if err := store.SaveArtifact(ctx, art); err != nil {
		t.Fatalf("SaveArtifact: %v", err)
	}

	got, err := store.GetArtifact(ctx, "art-1")
	if err != nil {
		t.Fatalf("GetArtifact: %v", err)
	}
	if string(got.Content) != string(art.Content) {
		t.Errorf("content mismatch: got %s", got.Content)
	}
}
