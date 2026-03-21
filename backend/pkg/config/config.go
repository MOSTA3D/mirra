package config

import "os"

// Config holds all application configuration loaded from environment variables.
// No hardcoded values — everything is configurable.
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	Env         string
	StorageBucket string
	StorageEndpoint string
	StorageKeyID string
	StorageKeySecret string
}

// Load reads configuration from environment variables with sensible defaults for local dev.
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		Env:              getEnv("ENV", "development"),
		StorageBucket:    getEnv("STORAGE_BUCKET", "mirra"),
		StorageEndpoint:  getEnv("STORAGE_ENDPOINT", ""),
		StorageKeyID:     getEnv("STORAGE_KEY_ID", ""),
		StorageKeySecret: getEnv("STORAGE_KEY_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
