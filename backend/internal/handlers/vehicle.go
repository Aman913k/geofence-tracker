package handlers

import (
	"net/http"

	"geofence-tracker/internal/models"
)

type createVehicleRequest struct {
	VehicleNumber string `json:"vehicle_number"`
	DriverName    string `json:"driver_name"`
	VehicleType   string `json:"vehicle_type"`
	Phone         string `json:"phone"`
}

// CreateVehicle handles POST /vehicles.
func (h *Handler) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req createVehicleRequest
	if !decodeJSON(ctx, w, r, &req) {
		return
	}

	switch {
	case req.VehicleNumber == "":
		respondError(ctx, w, http.StatusBadRequest, "vehicle_number is required")
		return
	case req.DriverName == "":
		respondError(ctx, w, http.StatusBadRequest, "driver_name is required")
		return
	case req.VehicleType == "":
		respondError(ctx, w, http.StatusBadRequest, "vehicle_type is required")
		return
	case req.Phone == "":
		respondError(ctx, w, http.StatusBadRequest, "phone is required")
		return
	}

	exists, err := h.Store.VehicleNumberExists(ctx, req.VehicleNumber)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to check vehicle: "+err.Error())
		return
	}
	if exists {
		respondError(ctx, w, http.StatusConflict, "a vehicle with this vehicle_number already exists")
		return
	}

	v := &models.Vehicle{
		ID:            newID("veh"),
		VehicleNumber: req.VehicleNumber,
		DriverName:    req.DriverName,
		VehicleType:   req.VehicleType,
		Phone:         req.Phone,
		Status:        "active",
	}
	if err := h.Store.CreateVehicle(ctx, v); err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to create vehicle: "+err.Error())
		return
	}

	respond(ctx, w, http.StatusCreated, map[string]any{
		"id":             v.ID,
		"vehicle_number": v.VehicleNumber,
		"status":         v.Status,
	})
}

// ListVehicles handles GET /vehicles.
func (h *Handler) ListVehicles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vehicles, err := h.Store.ListVehicles(ctx)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to list vehicles: "+err.Error())
		return
	}
	respond(ctx, w, http.StatusOK, map[string]any{"vehicles": vehicles})
}
