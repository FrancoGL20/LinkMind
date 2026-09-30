package service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/FrancoGL20/LinkMind/internal/domain"
)

// LinkRepository is the port that the service layer needs from the data layer.
// Defined here (in the consumer) so that the service package has zero knowledge
// of pgx, PostgreSQL, or any other storage technology — pure Clean Architecture.
// The concrete implementation lives in internal/repository and satisfies this
// interface implicitly (Go's structural typing — no "implements" keyword needed).
type LinkRepository interface {
	Create(ctx context.Context, link *domain.Link) (*domain.Link, error)
	FindByCode(ctx context.Context, code string) (*domain.Link, error)
	FindAll(ctx context.Context, limit, offset int) ([]*domain.Link, error)
	Count(ctx context.Context) (int64, error)
	Deactivate(ctx context.Context, code string) error
}

// ValidationError represents a user-facing input validation error.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// LinkService defines the business operations for link management.
type LinkService interface {
	Create(ctx context.Context, rawURL string) (*domain.Link, error)
	GetByCode(ctx context.Context, code string) (*domain.Link, error)
	List(ctx context.Context, page, limit int) (*ListResult, error)
	Deactivate(ctx context.Context, code string) error
}

// ListResult bundles a page of links with the pagination parameters that
// were actually applied (after clamping invalid input) and the total row
// count, so the handler can build accurate pagination metadata without
// repeating the business rules that produced page/limit.
type ListResult struct {
	Links []*domain.Link
	Page  int
	Limit int
	Total int64
}

// Pagination defaults and limits for LinkService.List — business rules that
// belong in the service layer, not in the handler or the repository.
const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

// linkService implements the LinkService interface.
// It depends on LinkRepository (interface defined above) — never on the concrete
// pgx implementation. This is the Dependency Inversion Principle in practice.
type linkService struct {
	repo LinkRepository
}

// NewLinkService creates a new linkService with any LinkRepository implementation.
// main.go passes *repository.linkRepository — Go verifies the interface match
// implicitly at compile time without this package ever importing "repository".
func NewLinkService(repo LinkRepository) LinkService {
	return &linkService{repo: repo}
}

// Create validates the URL and persists it to PostgreSQL via the repository.
//
// The code generation (strconv now, Base62 in step 2.1) happens inside
// the repository's Create method within a transaction. The service only
// owns the business rule: "valid URLs must be http or https".
func (s *linkService) Create(ctx context.Context, rawURL string) (*domain.Link, error) {
	// Validate URL format — this is a business rule, not a DB concern.
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, &ValidationError{Message: fmt.Sprintf("invalid URL: %q", rawURL)}
	}

	// Build the link stub. ID, Code, CreatedAt, and UpdatedAt are assigned by PostgreSQL.
	link := &domain.Link{
		LongURL:  rawURL,
		IsActive: true,
		AIStatus: domain.AIStatusPending,
	}

	// Persist to the database. The repository handles code generation atomically.
	created, err := s.repo.Create(ctx, link)
	if err != nil {
		return nil, fmt.Errorf("create link: %w", err)
	}

	return created, nil
}

// GetByCode retrieves an active link by its short code.
// The service delegates entirely to the repository — there is no additional
// business rule for a lookup. The domain.ErrNotFound sentinel is returned
// as-is so that the handler layer can distinguish 404 from 500.
func (s *linkService) GetByCode(ctx context.Context, code string) (*domain.Link, error) {
	link, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err // ErrNotFound or a real storage error — caller decides
	}
	return link, nil
}

// List returns a page of active links plus the total count needed to build
// pagination metadata (GET /api/links).
//
// Business rule: invalid or missing page/limit values fall back to sane
// defaults, and limit is capped at maxLimit so a client can never force the
// database to return an unbounded number of rows in one request. This
// clamping is a business decision, so it lives here — never in the handler
// (HTTP concerns only) nor in the repository (storage concerns only).
func (s *linkService) List(ctx context.Context, page, limit int) (*ListResult, error) {
	if page < 1 {
		page = defaultPage
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := (page - 1) * limit

	links, err := s.repo.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	// FindAll returns a nil slice when there are no rows — normalize to an
	// empty slice so the handler always serializes "data": [] instead of null.
	if links == nil {
		links = []*domain.Link{}
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("list links count: %w", err)
	}

	return &ListResult{Links: links, Page: page, Limit: limit, Total: total}, nil
}

// Deactivate performs a soft delete of a link by its short code.
// The service has no additional business rule beyond the repository's own
// guarantee (is_active = FALSE) — it exists so the handler never talks to
// the repository directly, keeping the dependency direction handler → service → repository.
func (s *linkService) Deactivate(ctx context.Context, code string) error {
	if err := s.repo.Deactivate(ctx, code); err != nil {
		return err // ErrNotFound or a real storage error — caller decides
	}
	return nil
}
