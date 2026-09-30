package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FrancoGL20/LinkMind/internal/domain"
	"github.com/FrancoGL20/LinkMind/internal/service"
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

// Count returns the total number of clicks recorded across all links.
// Used for the "total_clicks" field of GET /api/stats.
func (r *clickRepository) Count(ctx context.Context) (int64, error) {
	const query = `SELECT COUNT(*) FROM clicks`

	var total int64
	if err := r.pool.QueryRow(ctx, query).Scan(&total); err != nil {
		return 0, fmt.Errorf("click repository Count: %w", err)
	}
	return total, nil
}

// CountSince returns the number of clicks recorded on or after the given
// timestamp — used to compute "clicks_today" in GET /api/stats. The cutoff
// is calculated in Go for the same reason as LinkRepository.CountCreatedSince:
// the business defines "today", not the database session's timezone.
func (r *clickRepository) CountSince(ctx context.Context, since time.Time) (int64, error) {
	const query = `SELECT COUNT(*) FROM clicks WHERE clicked_at >= $1`

	var total int64
	if err := r.pool.QueryRow(ctx, query, since).Scan(&total); err != nil {
		return 0, fmt.Errorf("click repository CountSince: %w", err)
	}
	return total, nil
}

// TopLinks returns the N most-clicked active links, ordered by click count
// descending. It JOINs links and clicks — a link with zero clicks can never
// be "top", so an INNER JOIN (not LEFT JOIN) is the correct choice: excluding
// clickless links here is the intended business meaning, not a side effect
// to work around.
func (r *clickRepository) TopLinks(ctx context.Context, limit int) ([]service.TopLink, error) {
	const query = `
		SELECT l.code, COUNT(c.id) AS clicks
		FROM links l
		JOIN clicks c ON c.link_id = l.id
		WHERE l.is_active = TRUE
		GROUP BY l.code
		ORDER BY clicks DESC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("click repository TopLinks: %w", err)
	}
	defer rows.Close()

	// Pre-allocated with make(..., 0, limit) so the result is always a
	// non-nil, empty slice (never nil) when there are no clicks yet —
	// callers get a clean JSON "[]" instead of "null".
	top := make([]service.TopLink, 0, limit)
	for rows.Next() {
		var t service.TopLink
		if err := rows.Scan(&t.Code, &t.Clicks); err != nil {
			return nil, fmt.Errorf("click repository TopLinks scan: %w", err)
		}
		top = append(top, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("click repository TopLinks rows: %w", err)
	}
	return top, nil
}
