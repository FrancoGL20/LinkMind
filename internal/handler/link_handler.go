package handler

import (
	"encoding/json"
	"errors"
	"net/http"

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
		writeError(w, http.StatusInternalServerError, "failed to create link")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(link)
}

// writeError writes a JSON error response with the given status code and message.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
