package main

import (
	"context"
	"log"
	"net/http"

	"github.com/opengaebi/opengaebi/internal/api"
	"github.com/opengaebi/opengaebi/internal/config"
	"github.com/opengaebi/opengaebi/internal/db"
	"github.com/opengaebi/opengaebi/internal/registry"
)

func main() {
	cfg := config.Load()

	store, err := db.New(context.Background(), cfg.DBType, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer store.Close()

	regClient := registry.NewClient(cfg.RegistryURL, cfg.RegistryAPIKey)
	srv := api.New(store, cfg.APIKey, cfg.BaseURL, regClient)

	log.Printf("Opengaebi Bridge starting on :%d (db=%s)", cfg.Port, cfg.DBType)
	keyPreview := cfg.APIKey[:min(8, len(cfg.APIKey))]
	log.Printf("API Key: %s***", keyPreview)
	log.Printf("Base URL: %s", cfg.BaseURL)

	if err := http.ListenAndServe(srv.Addr(cfg.Port), srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

