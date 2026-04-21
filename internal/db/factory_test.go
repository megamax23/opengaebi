package db_test

import (
	"context"
	"testing"

	"github.com/opengaebi/opengaebi/internal/db"
)

func TestNew_SQLite(t *testing.T) {
	store, err := db.New(context.Background(), "sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New sqlite failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Error("expected non-nil DB")
	}
}

func TestNew_InvalidType(t *testing.T) {
	_, err := db.New(context.Background(), "mysql", "")
	if err == nil {
		t.Error("expected error for unsupported db type")
	}
}
