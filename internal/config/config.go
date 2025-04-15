package config

import (
	"os"
	"sync"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

var (
	config *Config
	once   sync.Once
)

func Get() *Config {
	once.Do(func() {
		// In Docker environments, use the service names defined in docker-compose.yml
		// For local development, default to localhost
		config = &Config{
			DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres123@postgres:5432/rtcs?sslmode=disable"),
			RedisURL:    getEnv("REDIS_URL", "redis://redis:6379/0"),
			JWTSecret:   getEnv("JWT_SECRET", "rtcs-secure-jwt-secret-key-2024"),
		}
	})
	return config
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
