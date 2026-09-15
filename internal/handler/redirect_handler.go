package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/FrancoGL20/LinkMind/internal/domain"
	"github.com/FrancoGL20/LinkMind/internal/service"
)

// RedirectHandler handles the public redirect endpoint GET /{code}.
// It lives in the handler layer: its only job is HTTP — read the code
// from the path, ask the service for the long URL, and write the response.
type RedirectHandler struct {
	svc service.LinkService
}

// NewRedirectHandler creates a RedirectHandler wired to the given LinkService.
func NewRedirectHandler(svc service.LinkService) *RedirectHandler {
	return &RedirectHandler{svc: svc}
}

// Redirect resolves a short code to its original URL and responds with HTTP 302.
//
// Flow:
//   GET /{code}
//     → read code from chi URL param
//     → ask service.GetByCode(ctx, code) → domain.Link
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

	link, err := h.svc.GetByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "short link not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, link.LongURL, http.StatusFound)
}
