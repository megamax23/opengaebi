package config_test

import (
	"os"
	"testing"

	"github.com/opengaebi/opengaebi/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("BRIDGE_DB")
	os.Unsetenv("BRIDGE_PORT")
	os.Unsetenv("BRIDGE_API_KEY")

	cfg := config.Load()

	if cfg.DBType != "sqlite" {
		t.Errorf("expected DBType=sqlite, got %s", cfg.DBType)
	}
	if cfg.Port != 7777 {
		t.Errorf("expected Port=7777, got %d", cfg.Port)
	}
	if cfg.APIKey == "" {
		t.Error("expected APIKey to be auto-generated, got empty")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("BRIDGE_DB", "postgres")
	os.Setenv("BRIDGE_PORT", "8888")
	os.Setenv("BRIDGE_API_KEY", "my-key")
	defer func() {
		os.Unsetenv("BRIDGE_DB")
		os.Unsetenv("BRIDGE_PORT")
		os.Unsetenv("BRIDGE_API_KEY")
	}()

	cfg := config.Load()

	if cfg.DBType != "postgres" {
		t.Errorf("expected DBType=postgres, got %s", cfg.DBType)
	}
	if cfg.Port != 8888 {
		t.Errorf("expected Port=8888, got %d", cfg.Port)
	}
	if cfg.APIKey != "my-key" {
		t.Errorf("expected APIKey=my-key, got %s", cfg.APIKey)
	}
}
