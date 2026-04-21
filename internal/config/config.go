package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DBType      string
	DatabaseURL string
	Port        int
	APIKey      string
	BaseURL     string
	RegistryURL string
}

func Load() Config {
	apiKey := getEnv("BRIDGE_API_KEY", "")
	if apiKey == "" {
		apiKey = generateAPIKey()
	}

	port := getEnvInt("BRIDGE_PORT", 7777)
	baseURL := getEnv("BRIDGE_BASE_URL", fmt.Sprintf("http://localhost:%d", port))

	return Config{
		DBType:      getEnv("BRIDGE_DB", "sqlite"),
		DatabaseURL: getEnv("DATABASE_URL", "./opengaebi.db"),
		Port:        port,
		APIKey:      apiKey,
		BaseURL:     baseURL,
		RegistryURL: getEnv("REGISTRY_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func generateAPIKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
