package handlers

import (
	"net/http"

	"geofence-tracker/internal/models"
)

type configureAlertRequest struct {
	GeofenceID string `json:"geofence_id"`
	VehicleID  string `json:"vehicle_id"`
	EventType  string `json:"event_type"`
}

// ConfigureAlert handles POST /alerts/configure.
func (h *Handler) ConfigureAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req configureAlertRequest
	if !decodeJSON(ctx, w, r, &req) {
		return
	}

	if req.GeofenceID == "" {
		respondError(ctx, w, http.StatusBadRequest, "geofence_id is required")
		return
	}
	if !models.ValidEventType(req.EventType) {
		respondError(ctx, w, http.StatusBadRequest, "event_type must be one of: entry, exit, both")
		return
	}

	// Referential checks give friendly 404s instead of a raw FK violation.
	if _, err := h.Store.GetGeofence(ctx, req.GeofenceID); err != nil {
		respondError(ctx, w, http.StatusNotFound, "geofence not found")
		return
	}
	if req.VehicleID != "" {
		if _, err := h.Store.GetVehicle(ctx, req.VehicleID); err != nil {
			respondError(ctx, w, http.StatusNotFound, "vehicle not found")
			return
		}
	}

	a := &models.AlertConfig{
		ID:         newID("alert"),
		GeofenceID: req.GeofenceID,
		VehicleID:  req.VehicleID,
		EventType:  req.EventType,
		Status:     "active",
	}
	if err := h.Store.CreateAlertConfig(ctx, a); err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to configure alert: "+err.Error())
		return
	}

	respond(ctx, w, http.StatusCreated, map[string]any{
		"alert_id":    a.ID,
		"geofence_id": a.GeofenceID,
		"vehicle_id":  a.VehicleID,
		"event_type":  a.EventType,
		"status":      a.Status,
	})
}

// ListAlerts handles GET /alerts (optional ?geofence_id= and ?vehicle_id= filters).
func (h *Handler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	alerts, err := h.Store.ListAlertConfigs(ctx, q.Get("geofence_id"), q.Get("vehicle_id"))
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to list alerts: "+err.Error())
		return
	}
	respond(ctx, w, http.StatusOK, map[string]any{"alerts": alerts})
}
