package domain

import "time"

// Click represents a single visit to a short link.
// IP addresses are stored as SHA-256 hashes for privacy by design.
type Click struct {
	ID        int64     `json:"id"`
	LinkID    int64     `json:"link_id"`
	ClickedAt time.Time `json:"clicked_at"`
	IPHash    string    `json:"ip_hash"`    // SHA-256 hash of the visitor's IP
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
}
