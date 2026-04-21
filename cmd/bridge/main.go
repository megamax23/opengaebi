package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/opengaebi/opengaebi/internal/config"
	"github.com/opengaebi/opengaebi/internal/db"
)

func main() {
	cfg := config.Load()

	store, err := db.New(context.Background(), cfg.DBType, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer store.Close()

	log.Printf("Opengaebi Bridge starting on :%d (db=%s)", cfg.Port, cfg.DBType)
	log.Printf("API Key: %s", cfg.APIKey)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
