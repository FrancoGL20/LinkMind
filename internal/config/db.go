package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDBPool creates a connection pool to PostgreSQL using pgxpool.
// It validates the DSN, configures pool limits, and verifies the connection
// with a Ping before returning the pool ready for use.
func NewDBPool(cfg *Config) (*pgxpool.Pool, error) {
	// Parse the DSN string into a pgxpool configuration struct.
	// This validates the format before attempting to connect.
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	// Configure pool behavior.
	poolCfg.MaxConns = 10                       // Max simultaneous connections
	poolCfg.MinConns = 2                        // Keep at least 2 idle connections ready
	poolCfg.MaxConnLifetime = 30 * time.Minute  // Recycle connections after 30 min
	poolCfg.MaxConnIdleTime = 5 * time.Minute   // Close idle connections after 5 min

	// Create the pool. This does NOT connect yet — pgxpool is lazy.
	// Connections are established on first query or explicit Ping.
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Ping verifies that the database is reachable and credentials are valid.
	// We use a 5-second timeout so it fails fast if the DB is down.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close() // Clean up the pool if ping fails
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
}
