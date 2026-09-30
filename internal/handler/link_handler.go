package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FrancoGL20/LinkMind/internal/domain"
	"github.com/FrancoGL20/LinkMind/internal/service"
)

// LinkHandler handles HTTP requests related to link operations.
type LinkHandler struct {
	linkService service.LinkService
}

// NewLinkHandler creates a new LinkHandler with the given LinkService.
func NewLinkHandler(ls service.LinkService) *LinkHandler {
	return &LinkHandler{linkService: ls}
}

// createLinkRequest is the expected JSON body for POST /api/links.
type createLinkRequest struct {
	URL string `json:"url"`
}

// Create handles POST /api/links.
// It decodes the request body, creates a shortened link, and responds with 201.
func (h *LinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url field is required")
		return
	}

	link, err := h.linkService.Create(r.Context(), req.URL)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			writeError(w, http.StatusUnprocessableEntity, validationErr.Error())
			return
		}
		log.Printf("ERROR create link: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create link")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(link)
}

// listLinksResponse is the JSON body for GET /api/links.
type listLinksResponse struct {
	Data       []*domain.Link `json:"data"`
	Pagination paginationMeta `json:"pagination"`
}

// paginationMeta describes the page that was actually returned — the values
// here are the ones LinkService.List applied after clamping, not necessarily
// the raw query params the client sent.
type paginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// List handles GET /api/links?page=&limit=.
// Reads pagination from query params and delegates all validation/clamping
// to the service layer — the handler's only job is HTTP in/out.
func (h *LinkHandler) List(w http.ResponseWriter, r *http.Request) {
	page := parseQueryInt(r, "page", 1)
	limit := parseQueryInt(r, "limit", 20)

	result, err := h.linkService.List(r.Context(), page, limit)
	if err != nil {
		log.Printf("ERROR list links: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list links")
		return
	}

	resp := listLinksResponse{
		Data: result.Links,
		Pagination: paginationMeta{
			Page:  result.Page,
			Limit: result.Limit,
			Total: result.Total,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Detail handles GET /api/links/{code} — returns the full record for a
// single active link, or 404 if the code doesn't exist or was deactivated.
func (h *LinkHandler) Detail(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	link, err := h.linkService.GetByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "link not found")
			return
		}
		log.Printf("ERROR get link detail: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get link")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(link)
}

// Delete handles DELETE /api/links/{code} — soft delete (is_active = false).
// Responds 204 No Content on success, matching the API contract: the
// resource still exists (for click history), but is no longer reachable.
func (h *LinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	if err := h.linkService.Deactivate(r.Context(), code); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "link not found")
			return
		}
		log.Printf("ERROR delete link: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete link")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseQueryInt reads an integer query parameter, falling back to a default
// value when it is missing or malformed. It only parses — range validation
// (min/max) is a business rule and belongs to the service layer.
func parseQueryInt(r *http.Request, key string, defaultValue int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return value
}

// writeError writes a JSON error response with the given status code and message.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
