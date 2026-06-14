package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"geofence-tracker/internal/db"
	"geofence-tracker/internal/handlers"
	"geofence-tracker/internal/middleware"
	"geofence-tracker/internal/ws"

	"github.com/gorilla/mux"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[geofence] ")

	// dsn := envOr("DATABASE_URL", "postgres://geofence:geofence@localhost:5432/geofence?sslmode=disable")
	dsn := envOr("DATABASE_URL", "postgresql://geofence_db_fmcm_user:BtSXXXChDbSi75TBDavKPJ0URPlBJQ0Z@dpg-d8nce14m0tmc73e0a0mg-a/geofence_db_fmcm")

	addr := envOr("LISTEN_ADDR", ":8080")

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := db.Connect(rootCtx, dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()
	log.Println("database connected and schema applied")

	hub := ws.NewHub()
	go hub.Run(rootCtx)

	h := handlers.New(store, hub)
	router := buildRouter(h, hub)

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-rootCtx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func buildRouter(h *handlers.Handler, hub *ws.Hub) http.Handler {
	r := mux.NewRouter()
	// r.Use(middleware.CORS)
	r.Use(middleware.Timing)

	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	// WebSocket alert stream (not wrapped by Timing — it's a long-lived upgrade).
	r.HandleFunc("/ws/alerts", hub.ServeWS)

	r.HandleFunc("/geofences", h.CreateGeofence).Methods(http.MethodPost)
	r.HandleFunc("/geofences", h.ListGeofences).Methods(http.MethodGet)
	r.HandleFunc("/geofences/{id}", h.GetGeofence).Methods(http.MethodGet)

	r.HandleFunc("/vehicles", h.CreateVehicle).Methods(http.MethodPost)
	r.HandleFunc("/vehicles", h.ListVehicles).Methods(http.MethodGet)

	// Order matters: the literal /vehicles/location must be registered before
	// the /vehicles/location/{vehicle_id} pattern is consulted for GETs.
	r.HandleFunc("/vehicles/location", h.UpdateLocation).Methods(http.MethodPost)
	r.HandleFunc("/vehicles/location/{vehicle_id}", h.GetVehicleLocation).Methods(http.MethodGet)

	r.HandleFunc("/alerts/configure", h.ConfigureAlert).Methods(http.MethodPost)
	r.HandleFunc("/alerts", h.ListAlerts).Methods(http.MethodGet)

	r.HandleFunc("/violations/history", h.ListViolations).Methods(http.MethodGet)

	// return r
	return middleware.CORS(r)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
