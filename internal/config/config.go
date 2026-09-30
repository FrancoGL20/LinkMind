package config

import (
	"os"
	"strconv"
)

// Config holds all configuration values loaded from environment variables.
// Use Load() to populate it at application startup.
type Config struct {
	// Server
	Port    string
	BaseURL string // Base URL used to build short URLs (e.g. "https://lnkm.io")

	// Database
	DatabaseURL string

	// Security
	APIKey     string
	IPHashSalt string // Secret salt for hashing visitor IPs before storing them

	// AI (Gemini)
	GeminiAPIKey string
	GeminiModel  string

	// Rate limiting
	RateLimitRPM int // Requests per minute per IP

	// Enrichment worker
	EnrichmentWorkers  int
	ScraperTimeoutSecs int
}

// Load reads configuration from environment variables and returns a Config.
// Missing optional fields fall back to sensible defaults.
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		APIKey:             getEnv("API_KEY", ""),
		IPHashSalt:         getEnv("IP_HASH_SALT", ""),
		GeminiAPIKey:       getEnv("GEMINI_API_KEY", ""),
		GeminiModel:        getEnv("GEMINI_MODEL", "gemini-1.5-flash"),
		RateLimitRPM:       getEnvInt("RATE_LIMIT_RPM", 60),
		EnrichmentWorkers:  getEnvInt("ENRICHMENT_WORKERS", 3),
		ScraperTimeoutSecs: getEnvInt("SCRAPER_TIMEOUT_SECS", 10),
	}
}

// getEnv returns the value of an environment variable, or a fallback default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns the integer value of an environment variable, or a fallback default.
func getEnvInt(key string, defaultValue int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return value
}
