package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FrancoGL20/LinkMind/internal/domain"
	"github.com/FrancoGL20/LinkMind/internal/service"
)

// linkRepository is the concrete PostgreSQL implementation that satisfies
// service.LinkRepository implicitly (Go's structural typing).
// It holds only a *pgxpool.Pool — no state, no mutexes needed (the pool is already safe
// for concurrent use from multiple goroutines).
type linkRepository struct {
	pool *pgxpool.Pool
}

// NewLinkRepository creates and returns a new linkRepository.
// Returns the concrete type — the interface (service.LinkRepository) is satisfied
// implicitly by Go at the call site in main.go. No import cycle needed.
func NewLinkRepository(pool *pgxpool.Pool) *linkRepository {
	return &linkRepository{pool: pool}
}

// Create inserts a link and sets its Base62 code within a single transaction.
//
// Why a transaction?
//   The code is derived from the BIGSERIAL id via Base62 encoding.
//   We can't compute the code before INSERT (we don't have the id yet).
//   Using a transaction guarantees that INSERT + UPDATE code happen atomically:
//   either both succeed, or neither does. No partial state is possible.
//
// Transaction flow:
//   BEGIN
//     INSERT INTO links (...) VALUES (...) RETURNING id  → get the id
//     code = Base62(id)                                  → compute short code
//     UPDATE links SET code = $code WHERE id = $id       → persist the code
//   COMMIT
//   Return the final link with the real Base62 code.
func (r *linkRepository) Create(ctx context.Context, link *domain.Link) (*domain.Link, error) {
	// Acquire a connection from the pool and begin a transaction.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	// Rollback is a no-op if Commit has already been called — always safe to defer.
	defer tx.Rollback(ctx)

	// Step 1: INSERT the link using a CTE to generate the sequence value exactly once.
	//
	// THE BUG WE'RE FIXING:
	//   The previous approach used VALUES (nextval('links_id_seq')::text, ...).
	//   PostgreSQL called nextval() for the 'code' column (consuming value N),
	//   then called nextval() AGAIN for the BIGSERIAL 'id' column (consuming N+1).
	//   Result: id = N+1, code = "N" — always different, always triggering an UPDATE.
	//   IDs skipped every other number (3, 5, 7...).
	//
	// THE FIX — CTE (Common Table Expression):
	//   WITH new_id AS (SELECT nextval('links_id_seq') AS val)
	//   This calls nextval() exactly once and stores the result in a temporary
	//   relation 'new_id'. We then SELECT from it to supply both 'id' and 'code',
	//   guaranteeing id == strconv(code) == the same sequence value.
	//
	//   Result: id = N, code = "N" → realCode == created.Code → UPDATE is skipped.
	//   IDs are now consecutive (1, 2, 3...).
	//
	// STEP 2.1 COMPATIBILITY:
	//   When Base62 is introduced, Base62(id) ≠ strconv(id), so the if-block below
	//   will correctly trigger the UPDATE with the real Base62 code. No changes needed here.
	const insertQuery = `
		WITH new_id AS (SELECT nextval('links_id_seq') AS val)
		INSERT INTO links (id, code, long_url, is_active, ai_status)
		SELECT val, val::text, $1, $2, $3 FROM new_id
		RETURNING id, code, long_url, title, summary, tags, is_active, ai_status, created_at, updated_at`

	row := tx.QueryRow(ctx, insertQuery,
		link.LongURL,
		link.IsActive,
		link.AIStatus,
	)

	created, err := scanLink(row)
	if err != nil {
		return nil, fmt.Errorf("create link insert: %w", err)
	}

	// Step 2: Compute the Base62 code from the database-assigned id.
	// This is the real short code that users will see in URLs (e.g. "gW3k").
	realCode := service.Encode(created.ID)

	// Step 3: Only update if the code needs to change.
	// With Base62: Encode(id) ≠ strconv(id) for all id > 0, so the UPDATE always runs.
	// This conditional is kept for correctness — it is a cheap string comparison.
	if realCode != created.Code {
		const updateQuery = `UPDATE links SET code = $1, updated_at = NOW() WHERE id = $2`
		_, err = tx.Exec(ctx, updateQuery, realCode, created.ID)
		if err != nil {
			return nil, fmt.Errorf("create link update code: %w", err)
		}
		created.Code = realCode
	}

	// Step 4: Commit the transaction. Both INSERT and UPDATE are now permanent.
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

// FindByCode retrieves an active link by its short code.
// Returns domain.ErrNotFound (a sentinel error) when no row matches.
//
// Why only active links?
//   Soft-deleted links (is_active=false) must not be accessible via redirect.
func (r *linkRepository) FindByCode(ctx context.Context, code string) (*domain.Link, error) {
	const query = `
		SELECT id, code, long_url, title, summary, tags, is_active, ai_status, created_at, updated_at
		FROM links
		WHERE code = $1 AND is_active = TRUE`

	row := r.pool.QueryRow(ctx, query, code)
	link, err := scanLink(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("link with code %q: %w", code, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("FindByCode query: %w", err)
	}
	return link, nil
}

// FindAll returns a paginated list of all active links, newest first.
//
// Pagination via LIMIT + OFFSET:
//   Simple and sufficient for this project. For very large datasets a
//   cursor-based approach (WHERE created_at < $cursor) would be more efficient,
//   but LIMIT/OFFSET is the standard starting point.
func (r *linkRepository) FindAll(ctx context.Context, limit, offset int) ([]*domain.Link, error) {
	const query = `
		SELECT id, code, long_url, title, summary, tags, is_active, ai_status, created_at, updated_at
		FROM links
		WHERE is_active = TRUE
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("FindAll query: %w", err)
	}
	defer rows.Close()

	var links []*domain.Link
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, fmt.Errorf("FindAll scan: %w", err)
		}
		links = append(links, link)
	}

	// rows.Err() must be checked after the loop — it captures errors that occurred
	// during iteration (e.g. network failure mid-stream), not just after the query.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindAll rows error: %w", err)
	}

	return links, nil
}

// scanLink is a helper that scans a pgx row (or rows.Next() row) into a domain.Link.
// Centralizing the scan logic here means that if we add a column to the struct,
// we only update this one function — not every query individually.
//
// The anonymous interface is satisfied by both *pgx.Row and pgx.Rows,
// so this helper works for QueryRow (single) and Query (multiple) results.
//
// Nullable columns (title, summary, tags) are scanned into pointer types first,
// then converted to Go zero values (empty string / nil slice) for the domain struct.
// pgx can scan a DB NULL into *string (sets it to nil), but NOT into string (panics).
func scanLink(row interface {
	Scan(dest ...any) error
}) (*domain.Link, error) {
	link := &domain.Link{}

	// Nullable intermediates for columns that allow NULL in PostgreSQL.
	// Title and Summary use pointers because string cannot be nil, but Tags is a slice which can be nil.
	var title, summary *string

	err := row.Scan(
		&link.ID,
		&link.Code,
		&link.LongURL,
		&title,     // nullable VARCHAR(512)
		&summary,   // nullable TEXT
		&link.Tags, // nullable TEXT[] — pgx returns nil slice for NULL
		&link.IsActive,
		&link.AIStatus,
		&link.CreatedAt,
		&link.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable pointers to Go zero values for the domain struct.
	// The domain doesn't need to distinguish between NULL and empty string.
	if title != nil {
		link.Title = *title
	}
	if summary != nil {
		link.Summary = *summary
	}

	return link, nil
}
