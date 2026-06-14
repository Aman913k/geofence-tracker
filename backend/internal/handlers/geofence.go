package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"geofence-tracker/internal/models"
)

type createGeofenceRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Coordinates []models.Coordinate `json:"coordinates"`
	Category    string              `json:"category"`
}

// validateCoordinates enforces the polygon rules from the spec.
func validateCoordinates(coords []models.Coordinate) string {
	if len(coords) < 4 {
		return "coordinates must contain at least 4 points (3 unique + 1 closing point)"
	}
	for _, c := range coords {
		if c.Lat() < -90 || c.Lat() > 90 {
			return "latitude must be between -90 and 90"
		}
		if c.Lng() < -180 || c.Lng() > 180 {
			return "longitude must be between -180 and 180"
		}
	}
	first, last := coords[0], coords[len(coords)-1]
	if first.Lat() != last.Lat() || first.Lng() != last.Lng() {
		return "first and last coordinates must be identical (closed polygon)"
	}
	return ""
}

// CreateGeofence handles POST /geofences.
func (h *Handler) CreateGeofence(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req createGeofenceRequest
	if !decodeJSON(ctx, w, r, &req) {
		return
	}

	if req.Name == "" {
		respondError(ctx, w, http.StatusBadRequest, "name is required")
		return
	}
	if !models.ValidCategory(req.Category) {
		respondError(ctx, w, http.StatusBadRequest, "category must be one of: delivery_zone, restricted_zone, toll_zone, customer_area")
		return
	}
	if msg := validateCoordinates(req.Coordinates); msg != "" {
		respondError(ctx, w, http.StatusBadRequest, msg)
		return
	}

	g := &models.Geofence{
		ID:          newID("geo"),
		Name:        req.Name,
		Description: req.Description,
		Coordinates: req.Coordinates,
		Category:    req.Category,
		Status:      "active",
	}
	if err := h.Store.CreateGeofence(ctx, g); err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to create geofence: "+err.Error())
		return
	}

	respond(ctx, w, http.StatusCreated, map[string]any{
		"id":     g.ID,
		"name":   g.Name,
		"status": g.Status,
	})
}

// ListGeofences handles GET /geofences (optional ?category= filter).
func (h *Handler) ListGeofences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	category := r.URL.Query().Get("category")
	if category != "" && !models.ValidCategory(category) {
		respondError(ctx, w, http.StatusBadRequest, "invalid category filter")
		return
	}

	geofences, err := h.Store.ListGeofences(ctx, category)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to list geofences: "+err.Error())
		return
	}
	respond(ctx, w, http.StatusOK, map[string]any{"geofences": geofences})
}

// GetGeofence handles GET /geofences/{id} (convenience endpoint).
func (h *Handler) GetGeofence(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	g, err := h.Store.GetGeofence(ctx, id)
	if err != nil {
		respondError(ctx, w, http.StatusNotFound, "geofence not found")
		return
	}
	respond(ctx, w, http.StatusOK, map[string]any{"geofence": g})
}
