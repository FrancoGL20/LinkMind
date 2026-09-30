package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/FrancoGL20/LinkMind/internal/service"
)

// StatsHandler handles HTTP requests for global system statistics.
type StatsHandler struct {
	statsService service.StatsService
}

// NewStatsHandler creates a new StatsHandler with the given StatsService.
func NewStatsHandler(ss service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: ss}
}

// GetStats handles GET /api/stats.
// There are no inputs to validate — this endpoint always returns the same
// shape of global metrics, so the handler is a thin pass-through to the
// service layer.
func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsService.GetGlobalStats(r.Context())
	if err != nil {
		log.Printf("ERROR get stats: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
