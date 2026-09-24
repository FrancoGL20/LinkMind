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
	"github.com/FrancoGL20/LinkMind/internal/repository"
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

	// Without a salt, an IP hash is reversible in minutes — warn but keep running,
	// since analytics must never block local development.
	if cfg.IPHashSalt == "" {
		log.Println("WARNING: IP_HASH_SALT is not set — visitor IP hashes are reversible")
	}

	// Create the PostgreSQL connection pool.
	pool, err := config.NewDBPool(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close() // Always close the pool when main exits.

	log.Printf("Database connected successfully (max_conns=%d)", pool.Config().MaxConns)

	// --- Dependency wiring (Clean Architecture: outer → inner) ---
	// The dependency graph flows from right to left:
	//   pool → linkRepo  → linkSvc  → linkHandler
	//   pool → clickRepo → clickSvc → redirectHandler
	// Each layer only knows about the layer immediately below it (via interface).
	// main is the only place where the concrete pgx types are visible.

	// Repository layer (data access — owns all SQL)
	linkRepo := repository.NewLinkRepository(pool)
	clickRepo := repository.NewClickRepository(pool)

	// Service layer (business logic — owns validation and orchestration)
	linkSvc := service.NewLinkService(linkRepo)
	clickSvc := service.NewClickService(clickRepo, cfg.IPHashSalt)

	// Handler layer (HTTP boundary — owns JSON encode/decode)
	linkHandler := handler.NewLinkHandler(linkSvc)
	redirectHandler := handler.NewRedirectHandler(linkSvc, clickSvc)

	// --- Router setup ---
	r := chi.NewRouter()

	// Built-in chi middleware: logs each request and recovers from panics.
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API routes (protected namespace — future auth middleware goes here)
	r.Post("/api/links", linkHandler.Create)

	// Public redirect route — must be outside /api to get a clean short URL.
	// Pattern: GET /{code} — chi captures everything after / as "code".
	// This route is registered LAST so that /api/... routes take priority.
	r.Get("/{code}", redirectHandler.Redirect)

	// --- Start HTTP server ---
	addr := ":" + cfg.Port
	log.Printf("LinkMind server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
