package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FrancoGL20/LinkMind/internal/domain"
)

// clickRepository is the concrete PostgreSQL implementation that satisfies
// service.ClickRepository implicitly (Go's structural typing).
// It holds only a *pgxpool.Pool — which is already safe for concurrent use, so
// no mutex is needed even though clicks are written from background goroutines.
type clickRepository struct {
	pool *pgxpool.Pool
}

// NewClickRepository creates and returns a new clickRepository.
// Returns the concrete type — the interface (service.ClickRepository) is
// satisfied implicitly by Go at the call site in main.go. No import cycle needed.
func NewClickRepository(pool *pgxpool.Pool) *clickRepository {
	return &clickRepository{pool: pool}
}

// Create inserts a single click row.
//
// Column decisions:
//   - id is omitted: the column is BIGSERIAL and the database owns the value.
//   - clicked_at IS sent from Go even though the column has DEFAULT NOW().
//     The insert runs in a background goroutine, so DEFAULT NOW() would record
//     when the write reached PostgreSQL, not when the visitor actually clicked.
//     The service captures the timestamp at redirect time and we persist that.
//   - ip_hash, user_agent and referer are nullable. pgx stores empty Go strings
//     as empty strings rather than NULL, which is fine for our analytics use case.
func (r *clickRepository) Create(ctx context.Context, click *domain.Click) error {
	const query = `
		INSERT INTO clicks (link_id, clicked_at, ip_hash, user_agent, referer)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.pool.Exec(ctx, query,
		click.LinkID,
		click.ClickedAt,
		click.IPHash,
		click.UserAgent,
		click.Referer,
	)
	if err != nil {
		return fmt.Errorf("click repository Create: %w", err)
	}
	return nil
}
