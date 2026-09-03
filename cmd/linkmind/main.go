package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/FrancoGL20/LinkMind/internal/config"
)

func main() {
	// Load .env file if it exists (development only).
	// In production (Railway), env vars are injected by the platform.
	if err := godotenv.Load(); err != nil {
		// Not an error — .env is optional in production.
		fmt.Fprintln(os.Stderr, "No .env file found, using system environment variables")
	}

	// Load all configuration from environment variables.
	cfg := config.Load()

	// Validate that DATABASE_URL is set — can't run without it.
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Create the PostgreSQL connection pool.
	pool, err := config.NewDBPool(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close() // Always close the pool when main exits.

	log.Printf("Database connected successfully (max_conns=%d)", pool.Config().MaxConns)
	log.Printf("LinkMind server starting on port %s", cfg.Port)
}
