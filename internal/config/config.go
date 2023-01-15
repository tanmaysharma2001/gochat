package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	Secret    []byte
	ExpiresIn time.Duration
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading .env file: %v", err)
	}

	return &Config{
		Server: ServerConfig{
			Port:         getEnvOrDefault("PORT", ":8080"),
			ReadTimeout:  getDurationOrDefault("READ_TIMEOUT", "15s"),
			WriteTimeout: getDurationOrDefault("WRITE_TIMEOUT", "15s"),
		},
		Database: DatabaseConfig{
			URL: getEnvOrDefault("DATABASE_URL", "postgres://chat:secret@localhost:5432/chatdb"),
		},
		JWT: JWTConfig{
			Secret:    []byte(getEnvOrFatal("JWT_SECRET")),
			ExpiresIn: getDurationOrDefault("JWT_EXPIRES_IN", "24h"),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return value
}

func getDurationOrDefault(key, defaultValue string) time.Duration {
	value := getEnvOrDefault(key, defaultValue)
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("Invalid duration for %s: %v", key, err)
	}
	return duration
}

func getIntOrDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("Invalid integer for %s: %v", key, err)
	}
	return intValue
}