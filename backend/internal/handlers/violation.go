package handlers

import (
	"net/http"
	"strconv"
	"time"

	"geofence-tracker/internal/db"
)

const (
	defaultViolationLimit = 50
	maxViolationLimit     = 500
)

// ListViolations handles GET /violations/history with optional filtering by
// vehicle, geofence, and date range, plus pagination via limit.
func (h *Handler) ListViolations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	filter := db.ViolationFilter{
		VehicleID:  q.Get("vehicle_id"),
		GeofenceID: q.Get("geofence_id"),
		Limit:      defaultViolationLimit,
	}

	if raw := q.Get("start_date"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			respondError(ctx, w, http.StatusBadRequest, "start_date must be ISO 8601 (RFC3339)")
			return
		}
		filter.StartDate = &t
	}
	if raw := q.Get("end_date"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			respondError(ctx, w, http.StatusBadRequest, "end_date must be ISO 8601 (RFC3339)")
			return
		}
		filter.EndDate = &t
	}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			respondError(ctx, w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if n > maxViolationLimit {
			n = maxViolationLimit
		}
		filter.Limit = n
	}

	violations, total, err := h.Store.ListViolations(ctx, filter)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to list violations: "+err.Error())
		return
	}

	respond(ctx, w, http.StatusOK, map[string]any{
		"violations":  violations,
		"total_count": total,
	})
}
