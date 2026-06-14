package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"geofence-tracker/internal/db"
	"geofence-tracker/internal/middleware"
	"geofence-tracker/internal/ws"
)

// Handler holds the dependencies shared by all HTTP handlers.
type Handler struct {
	Store *db.Store
	Hub   *ws.Hub
}

// New constructs a Handler.
func New(store *db.Store, hub *ws.Hub) *Handler {
	return &Handler{Store: store, Hub: hub}
}

// newID returns a prefixed, collision-resistant identifier, e.g. "geo_1a2b3c4d".
func newID(prefix string) string {
	return prefix + "_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
}

// respond writes payload as JSON with HTTP status, injecting the mandatory
// "time_ns" field (request execution time in nanoseconds) into the top-level
// object. Every successful API response flows through here so the field is
// never forgotten.
func respond(ctx context.Context, w http.ResponseWriter, status int, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "failed to encode response")
		return
	}

	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		respondError(ctx, w, http.StatusInternalServerError, "response was not a JSON object")
		return
	}
	obj["time_ns"], _ = json.Marshal(strconv.FormatInt(middleware.Elapsed(ctx), 10))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(obj); err != nil {
		log.Printf("respond: write failed: %v", err)
	}
}

// respondError writes a JSON error body (also carrying time_ns) with the status.
func respondError(ctx context.Context, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   message,
		"time_ns": strconv.FormatInt(middleware.Elapsed(ctx), 10),
	})
}

// decodeJSON parses the request body into dst, returning false (and writing a
// 400) if the body is missing or malformed.
func decodeJSON(ctx context.Context, w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		respondError(ctx, w, http.StatusBadRequest, "request body is required")
		return false
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		respondError(ctx, w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}
