package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"geofence-tracker/internal/models"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// polygonWKT renders [lat,lng] coordinate pairs as a PostGIS POLYGON literal.
// PostGIS expects "lng lat" (X Y) ordering, the reverse of the client format.
func polygonWKT(coords []models.Coordinate) string {
	parts := make([]string, len(coords))
	for i, c := range coords {
		parts[i] = fmt.Sprintf("%g %g", c.Lng(), c.Lat())
	}
	return fmt.Sprintf("POLYGON((%s))", strings.Join(parts, ", "))
}

// CreateGeofence persists a geofence, building its PostGIS polygon from coords.
func (s *Store) CreateGeofence(ctx context.Context, g *models.Geofence) error {
	coordsJSON, err := json.Marshal(g.Coordinates)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO geofences (id, name, description, category, coordinates, geom, status)
		VALUES ($1, $2, $3, $4, $5, ST_GeomFromText($6, 4326), $7)
		RETURNING created_at`
	return s.pool.QueryRow(ctx, q,
		g.ID, g.Name, g.Description, g.Category, coordsJSON, polygonWKT(g.Coordinates), g.Status,
	).Scan(&g.CreatedAt)
}

func scanGeofence(row pgx.Row) (models.Geofence, error) {
	var g models.Geofence
	var coordsJSON []byte
	if err := row.Scan(&g.ID, &g.Name, &g.Description, &g.Category, &coordsJSON, &g.Status, &g.CreatedAt); err != nil {
		return g, err
	}
	if err := json.Unmarshal(coordsJSON, &g.Coordinates); err != nil {
		return g, err
	}
	return g, nil
}

// ListGeofences returns all geofences, optionally filtered by category.
func (s *Store) ListGeofences(ctx context.Context, category string) ([]models.Geofence, error) {
	q := `SELECT id, name, description, category, coordinates, status, created_at FROM geofences`
	args := []any{}
	if category != "" {
		q += ` WHERE category = $1`
		args = append(args, category)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Geofence{}
	for rows.Next() {
		g, err := scanGeofence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GetGeofence returns a single geofence by id.
func (s *Store) GetGeofence(ctx context.Context, id string) (*models.Geofence, error) {
	const q = `SELECT id, name, description, category, coordinates, status, created_at FROM geofences WHERE id = $1`
	g, err := scanGeofence(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// CreateVehicle persists a new vehicle.
func (s *Store) CreateVehicle(ctx context.Context, v *models.Vehicle) error {
	const q = `
		INSERT INTO vehicles (id, vehicle_number, driver_name, vehicle_type, phone, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`
	return s.pool.QueryRow(ctx, q,
		v.ID, v.VehicleNumber, v.DriverName, v.VehicleType, v.Phone, v.Status,
	).Scan(&v.CreatedAt)
}

// VehicleNumberExists reports whether a vehicle with the given number is registered.
func (s *Store) VehicleNumberExists(ctx context.Context, number string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM vehicles WHERE vehicle_number = $1)`, number).Scan(&exists)
	return exists, err
}

// ListVehicles returns all registered vehicles.
func (s *Store) ListVehicles(ctx context.Context) ([]models.Vehicle, error) {
	const q = `SELECT id, vehicle_number, driver_name, vehicle_type, phone, status, created_at FROM vehicles ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Vehicle{}
	for rows.Next() {
		var v models.Vehicle
		if err := rows.Scan(&v.ID, &v.VehicleNumber, &v.DriverName, &v.VehicleType, &v.Phone, &v.Status, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetVehicle returns a single vehicle by id.
func (s *Store) GetVehicle(ctx context.Context, id string) (*models.Vehicle, error) {
	const q = `SELECT id, vehicle_number, driver_name, vehicle_type, phone, status, created_at FROM vehicles WHERE id = $1`
	var v models.Vehicle
	err := s.pool.QueryRow(ctx, q, id).Scan(&v.ID, &v.VehicleNumber, &v.DriverName, &v.VehicleType, &v.Phone, &v.Status, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// InsertLocation stores a vehicle location point.
func (s *Store) InsertLocation(ctx context.Context, vehicleID string, lat, lng float64, ts time.Time) error {
	const q = `
		INSERT INTO vehicle_locations (vehicle_id, latitude, longitude, geom, recorded_at)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($3, $2), 4326), $4)`
	_, err := s.pool.Exec(ctx, q, vehicleID, lat, lng, ts)
	return err
}

// LocationPoint is the most recent known position of a vehicle.
type LocationPoint struct {
	Latitude  float64
	Longitude float64
	Timestamp time.Time
}

// LatestLocation returns the most recent location for a vehicle, or nil if none.
func (s *Store) LatestLocation(ctx context.Context, vehicleID string) (*LocationPoint, error) {
	const q = `SELECT latitude, longitude, recorded_at FROM vehicle_locations WHERE vehicle_id = $1 ORDER BY recorded_at DESC, id DESC LIMIT 1`
	var p LocationPoint
	err := s.pool.QueryRow(ctx, q, vehicleID).Scan(&p.Latitude, &p.Longitude, &p.Timestamp)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GeofencesContaining returns every geofence whose polygon contains the point.
func (s *Store) GeofencesContaining(ctx context.Context, lat, lng float64) ([]models.Geofence, error) {
	const q = `
		SELECT id, name, description, category, coordinates, status, created_at
		FROM geofences
		WHERE ST_Contains(geom, ST_SetSRID(ST_MakePoint($2, $1), 4326))
		ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, q, lat, lng)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Geofence{}
	for rows.Next() {
		g, err := scanGeofence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// CurrentInsideSet returns the set of geofence ids the vehicle was last known inside.
func (s *Store) CurrentInsideSet(ctx context.Context, vehicleID string) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT geofence_id FROM vehicle_geofence_state WHERE vehicle_id = $1`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		set[id] = true
	}
	return set, rows.Err()
}

// ReplaceInsideSet overwrites the stored inside-set for a vehicle in one transaction.
func (s *Store) ReplaceInsideSet(ctx context.Context, vehicleID string, geofenceIDs []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM vehicle_geofence_state WHERE vehicle_id = $1`, vehicleID); err != nil {
		return err
	}
	for _, gid := range geofenceIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO vehicle_geofence_state (vehicle_id, geofence_id) VALUES ($1, $2)`,
			vehicleID, gid); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// InsertViolation records a geofence entry/exit event.
func (s *Store) InsertViolation(ctx context.Context, v *models.Violation) error {
	const q = `
		INSERT INTO violations (id, vehicle_id, geofence_id, event_type, latitude, longitude, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.pool.Exec(ctx, q, v.ID, v.VehicleID, v.GeofenceID, v.EventType, v.Latitude, v.Longitude, v.Timestamp)
	return err
}

// CreateAlertConfig persists an alert rule. An empty VehicleID is stored as NULL.
func (s *Store) CreateAlertConfig(ctx context.Context, a *models.AlertConfig) error {
	var vehicleID any
	if a.VehicleID != "" {
		vehicleID = a.VehicleID
	}
	const q = `
		INSERT INTO alert_configs (id, geofence_id, vehicle_id, event_type, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`
	return s.pool.QueryRow(ctx, q, a.ID, a.GeofenceID, vehicleID, a.EventType, a.Status).Scan(&a.CreatedAt)
}

// AlertConfigRow is an alert rule enriched with geofence and vehicle display fields.
type AlertConfigRow struct {
	models.AlertConfig
	GeofenceName  string `json:"geofence_name"`
	VehicleNumber string `json:"vehicle_number,omitempty"`
}

// ListAlertConfigs returns alert rules, optionally filtered by geofence and/or vehicle.
func (s *Store) ListAlertConfigs(ctx context.Context, geofenceID, vehicleID string) ([]AlertConfigRow, error) {
	q := `
		SELECT a.id, a.geofence_id, COALESCE(a.vehicle_id, ''), a.event_type, a.status, a.created_at,
		       g.name, COALESCE(v.vehicle_number, '')
		FROM alert_configs a
		JOIN geofences g ON g.id = a.geofence_id
		LEFT JOIN vehicles v ON v.id = a.vehicle_id
		WHERE 1=1`
	args := []any{}
	if geofenceID != "" {
		args = append(args, geofenceID)
		q += fmt.Sprintf(" AND a.geofence_id = $%d", len(args))
	}
	if vehicleID != "" {
		args = append(args, vehicleID)
		q += fmt.Sprintf(" AND a.vehicle_id = $%d", len(args))
	}
	q += ` ORDER BY a.created_at DESC`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AlertConfigRow{}
	for rows.Next() {
		var r AlertConfigRow
		if err := rows.Scan(&r.ID, &r.GeofenceID, &r.VehicleID, &r.EventType, &r.Status, &r.CreatedAt,
			&r.GeofenceName, &r.VehicleNumber); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MatchingAlertConfigs returns active alert rules that fire for the given event.
// A rule matches when its geofence matches, its event_type covers the event
// (exact match or "both"), and its vehicle is either unset (all vehicles) or
// equal to the given vehicle.
func (s *Store) MatchingAlertConfigs(ctx context.Context, geofenceID, vehicleID, eventType string) ([]models.AlertConfig, error) {
	const q = `
		SELECT id, geofence_id, COALESCE(vehicle_id, ''), event_type, status, created_at
		FROM alert_configs
		WHERE status = 'active'
		  AND geofence_id = $1
		  AND (event_type = $2 OR event_type = 'both')
		  AND (vehicle_id IS NULL OR vehicle_id = $3)`
	rows, err := s.pool.Query(ctx, q, geofenceID, eventType, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.AlertConfig{}
	for rows.Next() {
		var a models.AlertConfig
		if err := rows.Scan(&a.ID, &a.GeofenceID, &a.VehicleID, &a.EventType, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ViolationFilter narrows a violation-history query.
type ViolationFilter struct {
	VehicleID  string
	GeofenceID string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
}

// ListViolations returns violation history (with display names) plus the total
// count matching the filter (ignoring limit) for pagination.
func (s *Store) ListViolations(ctx context.Context, f ViolationFilter) ([]models.Violation, int, error) {
	where := " WHERE 1=1"
	args := []any{}
	add := func(cond string, val any) {
		args = append(args, val)
		where += fmt.Sprintf(" AND %s$%d", cond, len(args))
	}
	if f.VehicleID != "" {
		add("vi.vehicle_id = ", f.VehicleID)
	}
	if f.GeofenceID != "" {
		add("vi.geofence_id = ", f.GeofenceID)
	}
	if f.StartDate != nil {
		add("vi.recorded_at >= ", *f.StartDate)
	}
	if f.EndDate != nil {
		add("vi.recorded_at <= ", *f.EndDate)
	}

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM violations vi`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit)
	q := `
		SELECT vi.id, vi.vehicle_id, v.vehicle_number, vi.geofence_id, g.name,
		       vi.event_type, vi.latitude, vi.longitude, vi.recorded_at
		FROM violations vi
		JOIN vehicles v ON v.id = vi.vehicle_id
		JOIN geofences g ON g.id = vi.geofence_id` +
		where + fmt.Sprintf(" ORDER BY vi.recorded_at DESC LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []models.Violation{}
	for rows.Next() {
		var v models.Violation
		if err := rows.Scan(&v.ID, &v.VehicleID, &v.VehicleNumber, &v.GeofenceID, &v.GeofenceName,
			&v.EventType, &v.Latitude, &v.Longitude, &v.Timestamp); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}
