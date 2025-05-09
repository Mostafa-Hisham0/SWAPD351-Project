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

func isDocker() bool {
	return os.Getenv("DOCKER_ENV") == "true"
}

func Get() *Config {
	once.Do(func() {
		// Determine the database connection based on environment
		var dbURL string
		if isDocker() {
			dbURL = "postgres://postgres:postgres123@postgres:5432/rtcs?sslmode=disable"
		} else {
			dbURL = "postgres://postgres:wendy@localhost:5432/rtcs?sslmode=disable"
		}

		// Determine the Redis host based on environment
		redisHost := "localhost"
		if isDocker() {
			redisHost = "redis"
		}

		config = &Config{
			DatabaseURL: getEnv("DATABASE_URL", dbURL),
			RedisURL:    getEnv("REDIS_URL", "redis://"+redisHost+":6379/0"),
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
