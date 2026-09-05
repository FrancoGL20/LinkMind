package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/FrancoGL20/LinkMind/internal/config"
	"github.com/FrancoGL20/LinkMind/internal/handler"
	"github.com/FrancoGL20/LinkMind/internal/service"
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

	// --- Dependency wiring (Clean Architecture: outer → inner) ---
	// Service layer (business logic)
	linkSvc := service.NewLinkService()

	// Handler layer (HTTP boundary)
	linkHandler := handler.NewLinkHandler(linkSvc)

	// --- Router setup ---
	r := chi.NewRouter()

	// Built-in chi middleware: logs each request and recovers from panics.
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API routes
	r.Post("/api/links", linkHandler.Create)

	// --- Start HTTP server ---
	addr := ":" + cfg.Port
	log.Printf("LinkMind server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
