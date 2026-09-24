package handler

import (
	"errors"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/FrancoGL20/LinkMind/internal/domain"
	"github.com/FrancoGL20/LinkMind/internal/service"
)

// RedirectHandler handles the public redirect endpoint GET /{code}.
// It lives in the handler layer: its only job is HTTP — read the code from the
// path, ask the services for what it needs, and write the response.
//
// It depends on service interfaces only. Nothing here knows about pgx, SQL or
// PostgreSQL, so changing the storage engine never touches this file.
type RedirectHandler struct {
	linkSvc  service.LinkService
	clickSvc service.ClickService
}

// NewRedirectHandler creates a RedirectHandler wired to the given services.
func NewRedirectHandler(linkSvc service.LinkService, clickSvc service.ClickService) *RedirectHandler {
	return &RedirectHandler{linkSvc: linkSvc, clickSvc: clickSvc}
}

// Redirect resolves a short code to its original URL and responds with HTTP 302.
//
// Flow:
//   GET /{code}
//     → read code from chi URL param
//     → ask service.GetByCode(ctx, code) → domain.Link
//     → clickSvc.RecordAsync(...)  ← returns immediately, writes in background
//     → 302 Location: link.LongURL
//
// Error cases:
//   - code not found or inactive → 404 plain text
//   - any other error            → 500 plain text
//
// Why 302 (temporary) instead of 301 (permanent)?
//   301 is cached aggressively by browsers and CDNs. Once a browser caches
//   a 301, it will never hit our server again for that code, even if we later
//   change the destination URL or deactivate the link. 302 ensures every
//   redirect goes through our server, which lets us count clicks accurately
//   and react to changes immediately.
func (h *RedirectHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	link, err := h.linkSvc.GetByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "short link not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Read the request metadata here, before responding: once this handler
	// returns, the server may recycle the *http.Request for the next connection.
	h.clickSvc.RecordAsync(link.ID, clientIP(r), r.UserAgent(), r.Referer())

	http.Redirect(w, r, link.LongURL, http.StatusFound)
}

// clientIP extracts the visitor's IP address from the request.
//
// r.RemoteAddr carries "host:port", and the port is an ephemeral value the OS
// picks per TCP connection — it changes on every single request. Hashing it
// would produce a different digest for the same visitor every time, which makes
// unique-visitor metrics meaningless. The port must be stripped first.
//
// Once deployed behind a proxy (Railway, in phase 4), RemoteAddr will be the
// proxy's address and the real client IP will arrive in X-Forwarded-For. That
// header is only trustworthy behind a known proxy, so it is handled at that
// point with chi's middleware.RealIP rather than parsed blindly here.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr is not always host:port (e.g. Unix sockets) — use it as is.
		return r.RemoteAddr
	}
	return host
}
