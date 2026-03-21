package config

import "os"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port             string
	DBDriver         string // memory | postgres
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	Env              string
	StorageBucket    string
	StorageEndpoint  string
	StorageKeyID     string
	StorageKeySecret string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		DBDriver:         getEnv("DB_DRIVER", "memory"),
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:        getEnv("JWT_SECRET", "dev-secret-change-in-prod"),
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
