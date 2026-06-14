package models

import "time"

// Geofence categories.
const (
	CategoryDeliveryZone   = "delivery_zone"
	CategoryRestrictedZone = "restricted_zone"
	CategoryTollZone       = "toll_zone"
	CategoryCustomerArea   = "customer_area"
)

// Alert / event types.
const (
	EventEntry = "entry"
	EventExit  = "exit"
	EventBoth  = "both"
)

// ValidCategories reports whether c is an accepted geofence category.
func ValidCategory(c string) bool {
	switch c {
	case CategoryDeliveryZone, CategoryRestrictedZone, CategoryTollZone, CategoryCustomerArea:
		return true
	}
	return false
}

// ValidEventType reports whether e is an accepted alert event type.
func ValidEventType(e string) bool {
	switch e {
	case EventEntry, EventExit, EventBoth:
		return true
	}
	return false
}

// Coordinate is a [latitude, longitude] pair as sent by clients.
type Coordinate [2]float64

func (c Coordinate) Lat() float64 { return c[0] }
func (c Coordinate) Lng() float64 { return c[1] }

// Geofence is a stored polygonal boundary.
type Geofence struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Coordinates []Coordinate `json:"coordinates"`
	Category    string       `json:"category"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
}

// Vehicle is a registered, trackable vehicle.
type Vehicle struct {
	ID            string    `json:"id"`
	VehicleNumber string    `json:"vehicle_number"`
	DriverName    string    `json:"driver_name"`
	VehicleType   string    `json:"vehicle_type"`
	Phone         string    `json:"phone"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// AlertConfig is a rule describing which geofence events should raise alerts.
type AlertConfig struct {
	ID        string    `json:"alert_id"`
	GeofenceID string   `json:"geofence_id"`
	VehicleID string    `json:"vehicle_id,omitempty"`
	EventType string    `json:"event_type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Violation is a recorded entry/exit event for a vehicle against a geofence.
type Violation struct {
	ID            string    `json:"id"`
	VehicleID     string    `json:"vehicle_id"`
	VehicleNumber string    `json:"vehicle_number"`
	GeofenceID    string    `json:"geofence_id"`
	GeofenceName  string    `json:"geofence_name"`
	EventType     string    `json:"event_type"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Timestamp     time.Time `json:"timestamp"`
}
