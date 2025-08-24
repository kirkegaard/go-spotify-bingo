package config

import (
	"os"
)

type Config struct {
	Port          string
	SpotifyID     string
	SpotifySecret string
	BaseURL       string
	DatabasePath  string
	SessionSecret string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		SpotifyID:     getEnv("SPOTIFY_CLIENT_ID", ""),
		SpotifySecret: getEnv("SPOTIFY_CLIENT_SECRET", ""),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		DatabasePath:  getEnv("DATABASE_PATH", "./bingo.db"),
		SessionSecret: getEnv("SESSION_SECRET", "your-secret-key-here"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
