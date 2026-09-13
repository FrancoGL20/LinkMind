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
}

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
