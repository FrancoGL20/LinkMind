package service

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/FrancoGL20/LinkMind/internal/domain"
)

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
type linkService struct {
	// idCounter is a temporary in-memory ID generator.
	// It will be replaced by the database BIGSERIAL in step 1.7.
	idCounter atomic.Int64
}

// NewLinkService creates a new linkService instance.
func NewLinkService() LinkService {
	return &linkService{}
}

// Create validates the URL and returns a new shortened Link.
// NOTE: This is a temporary implementation without database persistence.
// The link is only stored in memory and lost on restart (fixed in step 1.7).
func (s *linkService) Create(ctx context.Context, rawURL string) (*domain.Link, error) {
	// Validate that the URL is well-formed.
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, &ValidationError{Message: fmt.Sprintf("invalid URL: %q", rawURL)}
	}

	// Generate a temporary sequential ID (replaced by DB BIGSERIAL in step 1.7).
	id := s.idCounter.Add(1)
	now := time.Now().UTC()

	link := &domain.Link{
		ID:        id,
		Code:      strconv.FormatInt(id, 10), // Temporary: code = ID as string (replaced by Base62 in step 2.1)
		LongURL:   rawURL,
		IsActive:  true,
		AIStatus:  domain.AIStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return link, nil
}
