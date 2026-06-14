package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"geofence-tracker/internal/models"
)

type updateLocationRequest struct {
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp string  `json:"timestamp"`
}

// alertMessage is the real-time payload broadcast over WebSocket and the shape
// described in the spec for /ws/alerts.
type alertMessage struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Timestamp string `json:"timestamp"`
	Vehicle   struct {
		VehicleID     string `json:"vehicle_id"`
		VehicleNumber string `json:"vehicle_number"`
		DriverName    string `json:"driver_name"`
	} `json:"vehicle"`
	Geofence struct {
		GeofenceID   string `json:"geofence_id"`
		GeofenceName string `json:"geofence_name"`
		Category     string `json:"category"`
	} `json:"geofence"`
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
}

type geofenceStatus struct {
	GeofenceID   string `json:"geofence_id"`
	GeofenceName string `json:"geofence_name"`
	Status       string `json:"status"`
}

// UpdateLocation handles POST /vehicles/location: it stores the position,
// determines which geofences contain it, detects entry/exit transitions versus
// the vehicle's last known state, records those events, and fires real-time
// alerts for any matching alert rules — the alert delivery happening
// asynchronously so the HTTP response is not blocked.
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req updateLocationRequest
	if !decodeJSON(ctx, w, r, &req) {
		return
	}

	if req.VehicleID == "" {
		respondError(ctx, w, http.StatusBadRequest, "vehicle_id is required")
		return
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		respondError(ctx, w, http.StatusBadRequest, "latitude must be between -90 and 90")
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		respondError(ctx, w, http.StatusBadRequest, "longitude must be between -180 and 180")
		return
	}
	if req.Timestamp == "" {
		respondError(ctx, w, http.StatusBadRequest, "timestamp is required")
		return
	}
	ts, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		respondError(ctx, w, http.StatusBadRequest, "timestamp must be ISO 8601 (RFC3339), e.g. 2025-01-15T10:35:00Z")
		return
	}

	vehicle, err := h.Store.GetVehicle(ctx, req.VehicleID)
	if err != nil {
		respondError(ctx, w, http.StatusNotFound, "vehicle not found")
		return
	}

	if err := h.Store.InsertLocation(ctx, vehicle.ID, req.Latitude, req.Longitude, ts); err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to store location: "+err.Error())
		return
	}

	// Geofences currently containing the point.
	current, err := h.Store.GeofencesContaining(ctx, req.Latitude, req.Longitude)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to evaluate geofences: "+err.Error())
		return
	}

	// Previous inside-set to diff against for entry/exit detection.
	prev, err := h.Store.CurrentInsideSet(ctx, vehicle.ID)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to load prior state: "+err.Error())
		return
	}

	currentByID := make(map[string]models.Geofence, len(current))
	statuses := make([]geofenceStatus, 0, len(current))
	currentIDs := make([]string, 0, len(current))
	for _, g := range current {
		currentByID[g.ID] = g
		currentIDs = append(currentIDs, g.ID)
		statuses = append(statuses, geofenceStatus{GeofenceID: g.ID, GeofenceName: g.Name, Status: "inside"})
	}

	// Process transitions and alert delivery off the request path.
	h.processTransitions(vehicle, prev, currentByID, currentIDs, req.Latitude, req.Longitude, ts)

	respond(ctx, w, http.StatusOK, map[string]any{
		"vehicle_id":        vehicle.ID,
		"location_updated":  true,
		"current_geofences": statuses,
	})
}

// processTransitions diffs the previous and current inside-sets, records each
// entry/exit as a violation, fires matching alerts, and persists the new state.
// It runs in its own goroutine with a detached context so the HTTP response is
// not blocked by alert delivery (spec requirement).
func (h *Handler) processTransitions(
	vehicle *models.Vehicle,
	prev map[string]bool,
	currentByID map[string]models.Geofence,
	currentIDs []string,
	lat, lng float64,
	ts time.Time,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Entries: in current set but not previously inside.
		for id, g := range currentByID {
			if !prev[id] {
				h.recordAndAlert(ctx, vehicle, g, models.EventEntry, lat, lng, ts)
			}
		}
		// Exits: previously inside but no longer in the current set.
		for id := range prev {
			if _, stillInside := currentByID[id]; !stillInside {
				g, err := h.Store.GetGeofence(ctx, id)
				if err != nil {
					log.Printf("transition: load exited geofence %s: %v", id, err)
					continue
				}
				h.recordAndAlert(ctx, vehicle, *g, models.EventExit, lat, lng, ts)
			}
		}

		if err := h.Store.ReplaceInsideSet(ctx, vehicle.ID, currentIDs); err != nil {
			log.Printf("transition: persist inside-set for %s: %v", vehicle.ID, err)
		}
	}()
}

// recordAndAlert stores a violation for the event and broadcasts an alert if any
// active alert rule matches.
func (h *Handler) recordAndAlert(
	ctx context.Context,
	vehicle *models.Vehicle,
	g models.Geofence,
	eventType string,
	lat, lng float64,
	ts time.Time,
) {
	viol := &models.Violation{
		ID:         newID("viol"),
		VehicleID:  vehicle.ID,
		GeofenceID: g.ID,
		EventType:  eventType,
		Latitude:   lat,
		Longitude:  lng,
		Timestamp:  ts,
	}
	if err := h.Store.InsertViolation(ctx, viol); err != nil {
		log.Printf("transition: insert violation: %v", err)
	}

	matches, err := h.Store.MatchingAlertConfigs(ctx, g.ID, vehicle.ID, eventType)
	if err != nil {
		log.Printf("transition: match alert configs: %v", err)
		return
	}
	if len(matches) == 0 {
		return
	}

	var msg alertMessage
	msg.EventID = newID("evt")
	msg.EventType = eventType
	msg.Timestamp = ts.UTC().Format(time.RFC3339)
	msg.Vehicle.VehicleID = vehicle.ID
	msg.Vehicle.VehicleNumber = vehicle.VehicleNumber
	msg.Vehicle.DriverName = vehicle.DriverName
	msg.Geofence.GeofenceID = g.ID
	msg.Geofence.GeofenceName = g.Name
	msg.Geofence.Category = g.Category
	msg.Location.Latitude = lat
	msg.Location.Longitude = lng

	h.Hub.Broadcast(msg)
}

// GetVehicleLocation handles GET /vehicles/location/{vehicle_id}.
func (h *Handler) GetVehicleLocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vehicleID := mux.Vars(r)["vehicle_id"]

	vehicle, err := h.Store.GetVehicle(ctx, vehicleID)
	if err != nil {
		respondError(ctx, w, http.StatusNotFound, "vehicle not found")
		return
	}

	loc, err := h.Store.LatestLocation(ctx, vehicle.ID)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to load location: "+err.Error())
		return
	}

	resp := map[string]any{
		"vehicle_id":        vehicle.ID,
		"vehicle_number":    vehicle.VehicleNumber,
		"current_location":  nil,
		"current_geofences": []any{},
	}

	if loc != nil {
		resp["current_location"] = map[string]any{
			"latitude":  loc.Latitude,
			"longitude": loc.Longitude,
			"timestamp": loc.Timestamp.UTC().Format(time.RFC3339),
		}
		inside, err := h.Store.GeofencesContaining(ctx, loc.Latitude, loc.Longitude)
		if err != nil {
			respondError(ctx, w, http.StatusInternalServerError, "failed to evaluate geofences: "+err.Error())
			return
		}
		fences := make([]map[string]any, 0, len(inside))
		for _, g := range inside {
			fences = append(fences, map[string]any{
				"geofence_id":   g.ID,
				"geofence_name": g.Name,
				"category":      g.Category,
			})
		}
		resp["current_geofences"] = fences
	}

	respond(ctx, w, http.StatusOK, resp)
}
