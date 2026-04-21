package db_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

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

func TestPostgres_SendAndPollMessages(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	store, err := db.NewPostgres(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	id := "msg-pg-" + fmt.Sprintf("%d", time.Now().UnixNano())
	msg := db.Message{ID: id, FromPeer: "a", ToPeer: "pg-b", Workspace: "pgtest", Payload: "hello-pg"}

	if err := store.SendMessage(ctx, msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	msgs, err := store.PollMessages(ctx, "pgtest", "pg-b", 10)
	if err != nil {
		t.Fatalf("PollMessages: %v", err)
	}
	found := false
	for _, m := range msgs {
		if m.ID == id {
			found = true
		}
	}
	if !found {
		t.Error("sent message not found in poll result")
	}
	store.DeleteMessage(ctx, id)
}

func TestPostgres_SaveAndGetArtifact(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	store, err := db.NewPostgres(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	id := "art-pg-" + fmt.Sprintf("%d", time.Now().UnixNano())
	art := db.Artifact{ID: id, Workspace: "pgtest", Name: "test.txt", Kind: "text", Content: []byte("hello")}

	if err := store.SaveArtifact(ctx, art); err != nil {
		t.Fatalf("SaveArtifact: %v", err)
	}
	got, err := store.GetArtifact(ctx, id)
	if err != nil {
		t.Fatalf("GetArtifact: %v", err)
	}
	if string(got.Content) != "hello" {
		t.Errorf("content mismatch: %s", got.Content)
	}
}
