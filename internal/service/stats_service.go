package service

import (
	"context"
	"fmt"
	"time"
)

// topLinksLimit bounds how many entries appear in the "top_links" ranking of
// GET /api/stats. A business rule, so it lives in the service layer.
const topLinksLimit = 5

// TopLink represents a single entry in the "most visited links" ranking.
// Defined here (service layer), not in domain, because it is not a core
// entity — it is a reporting shape produced by a repository method that
// JOINs links and clicks (see ClickRepository.TopLinks). The domain package
// should have zero knowledge of aggregation queries.
type TopLink struct {
	Code   string `json:"code"`
	Clicks int64  `json:"clicks"`
}

// Stats aggregates the global metrics exposed by GET /api/stats.
type Stats struct {
	TotalLinks  int64     `json:"total_links"`
	TotalClicks int64     `json:"total_clicks"`
	LinksToday  int64     `json:"links_today"`
	ClicksToday int64     `json:"clicks_today"`
	TopLinks    []TopLink `json:"top_links"`
}

// statsClickRepository is the narrow slice of ClickRepository's storage
// capabilities that StatsService actually needs. It is declared separately
// from ClickRepository (which only exposes Create, for click_service.go)
// following the Interface Segregation Principle: StatsService never calls
// Create, so it should not depend on it. *repository.clickRepository
// satisfies this interface implicitly — it already implements all three.
type statsClickRepository interface {
	Count(ctx context.Context) (int64, error)
	CountSince(ctx context.Context, since time.Time) (int64, error)
	TopLinks(ctx context.Context, limit int) ([]TopLink, error)
}

// statsLinkRepository is the narrow slice of link storage capabilities that
// StatsService needs — same Interface Segregation reasoning as statsClickRepository
// above. *repository.linkRepository satisfies this interface implicitly — it
// already implements both methods.
type statsLinkRepository interface {
	Count(ctx context.Context) (int64, error)
	CountCreatedSince(ctx context.Context, since time.Time) (int64, error)
}

// StatsService defines the reporting operation exposed by GET /api/stats.
type StatsService interface {
	GetGlobalStats(ctx context.Context) (*Stats, error)
}

// statsService implements StatsService. It depends only on the read-side
// ports of LinkRepository and ClickRepository — never on pgx directly.
type statsService struct {
	linkRepo  statsLinkRepository
	clickRepo statsClickRepository
}

// NewStatsService creates a statsService wired to the link and click
// repositories. main.go passes the same *repository.linkRepository and
// *repository.clickRepository instances used by LinkService and ClickService —
// one PostgreSQL table, one repository implementation, reused by every
// service that needs it.
func NewStatsService(linkRepo statsLinkRepository, clickRepo statsClickRepository) StatsService {
	return &statsService{linkRepo: linkRepo, clickRepo: clickRepo}
}

// GetGlobalStats computes the five metrics of GET /api/stats.
//
// "Today" is calculated once, in Go, as midnight UTC of the current day —
// not with PostgreSQL's CURRENT_DATE — so the business defines what "today"
// means regardless of the database server's session timezone.
func (s *statsService) GetGlobalStats(ctx context.Context) (*Stats, error) {
	since := time.Now().Truncate(24 * time.Hour)

	totalLinks, err := s.linkRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats total links: %w", err)
	}

	totalClicks, err := s.clickRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats total clicks: %w", err)
	}

	linksToday, err := s.linkRepo.CountCreatedSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("stats links today: %w", err)
	}

	clicksToday, err := s.clickRepo.CountSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("stats clicks today: %w", err)
	}

	topLinks, err := s.clickRepo.TopLinks(ctx, topLinksLimit)
	if err != nil {
		return nil, fmt.Errorf("stats top links: %w", err)
	}

	return &Stats{
		TotalLinks:  totalLinks,
		TotalClicks: totalClicks,
		LinksToday:  linksToday,
		ClicksToday: clicksToday,
		TopLinks:    topLinks,
	}, nil
}
