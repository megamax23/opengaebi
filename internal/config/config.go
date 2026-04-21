package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strconv"
)

type Config struct {
	DBType      string
	DatabaseURL string
	Port        int
	APIKey      string
	RegistryURL string
}

func Load() Config {
	apiKey := getEnv("BRIDGE_API_KEY", "")
	if apiKey == "" {
		apiKey = generateAPIKey()
	}

	return Config{
		DBType:      getEnv("BRIDGE_DB", "sqlite"),
		DatabaseURL: getEnv("DATABASE_URL", "./opengaebi.db"),
		Port:        getEnvInt("BRIDGE_PORT", 7777),
		APIKey:      apiKey,
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
