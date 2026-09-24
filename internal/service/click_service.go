package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	"github.com/FrancoGL20/LinkMind/internal/domain"
)

// ClickRepository is the port that the service layer needs from the data layer.
// Defined here (in the consumer) so that the service package has zero knowledge
// of pgx, PostgreSQL, or any other storage technology — same pattern as
// LinkRepository. The concrete implementation lives in internal/repository and
// satisfies this interface implicitly.
type ClickRepository interface {
	Create(ctx context.Context, click *domain.Click) error
}

// ClickService defines the analytics operations triggered by a redirect.
type ClickService interface {
	// RecordAsync persists a click in the background. It returns immediately and
	// never blocks the caller, so analytics can never delay a user's redirect.
	RecordAsync(linkID int64, ip, userAgent, referer string)
}

// clickWriteTimeout bounds the lifetime of a background insert so that a slow
// database can never leak goroutines indefinitely.
const clickWriteTimeout = 5 * time.Second

// clickService implements ClickService. It owns the three policy decisions that
// surround click persistence: when the click happened, how long we are willing
// to wait for the write, and what to do when the write fails.
type clickService struct {
	repo ClickRepository
	salt string
}

// NewClickService creates a clickService. The salt is used to hash visitor IPs.
func NewClickService(repo ClickRepository, ipHashSalt string) ClickService {
	return &clickService{repo: repo, salt: ipHashSalt}
}

// RecordAsync builds the Click and hands the insert to a background goroutine.
//
// The Click is built on the caller's goroutine on purpose: ClickedAt must be the
// moment the redirect happened, not the moment the scheduler happens to run the
// goroutine. Only the database round trip is deferred.
//
// The goroutine is fire-and-forget by design: a failed analytics write must never
// affect the redirect, which has already been sent by the time this runs.
func (s *clickService) RecordAsync(linkID int64, ip, userAgent, referer string) {
	click := &domain.Click{
		LinkID:    linkID,
		ClickedAt: time.Now(),
		IPHash:    s.hashIP(ip),
		UserAgent: userAgent,
		Referer:   referer,
	}

	go func() {
		// The request context is already cancelled once the handler returns, so
		// the background write starts from Background() with its own deadline.
		ctx, cancel := context.WithTimeout(context.Background(), clickWriteTimeout)
		defer cancel()

		if err := s.repo.Create(ctx, click); err != nil {
			log.Printf("ERROR record click: link_id=%d: %v", linkID, err)
		}
	}()
}

// hashIP returns the salted SHA-256 digest of a visitor's IP address.
//
// Why the salt matters:
//   An unsalted SHA-256 of an IPv4 address is not anonymisation. The whole
//   address space is only 2^32 values, so the full rainbow table can be
//   precomputed in minutes and every hash reversed. A secret salt keeps the
//   digest stable (same visitor always produces the same hash, which is what
//   unique-visitor metrics need) while making it practically irreversible.
func (s *clickService) hashIP(ip string) string {
	sum := sha256.Sum256([]byte(s.salt + ip))
	return hex.EncodeToString(sum[:])
}
