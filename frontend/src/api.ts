import type {
  AlertConfig,
  Category,
  Coordinate,
  EventType,
  Geofence,
  Vehicle,
  VehicleLocation,
  Violation,
} from "./types";

const BASE = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  const text = await res.text();
  const body = text ? JSON.parse(text) : {};
  if (!res.ok) {
    throw new Error(body.error || `Request failed (${res.status})`);
  }
  return body as T;
}

// ---- Geofences ----

export function listGeofences(category?: Category) {
  const q = category ? `?category=${encodeURIComponent(category)}` : "";
  return request<{ geofences: Geofence[] }>(`/geofences${q}`).then(
    (r) => r.geofences,
  );
}

export function createGeofence(input: {
  name: string;
  description: string;
  category: Category;
  coordinates: Coordinate[];
}) {
  return request<{ id: string; name: string; status: string }>("/geofences", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

// ---- Vehicles ----

export function listVehicles() {
  return request<{ vehicles: Vehicle[] }>("/vehicles").then((r) => r.vehicles);
}

export function createVehicle(input: {
  vehicle_number: string;
  driver_name: string;
  vehicle_type: string;
  phone: string;
}) {
  return request<{ id: string; vehicle_number: string; status: string }>(
    "/vehicles",
    { method: "POST", body: JSON.stringify(input) },
  );
}

export function getVehicleLocation(vehicleId: string) {
  return request<VehicleLocation>(
    `/vehicles/location/${encodeURIComponent(vehicleId)}`,
  );
}

export function updateLocation(input: {
  vehicle_id: string;
  latitude: number;
  longitude: number;
  timestamp: string;
}) {
  return request<{
    vehicle_id: string;
    location_updated: boolean;
    current_geofences: { geofence_id: string; geofence_name: string; status: string }[];
  }>("/vehicles/location", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

// ---- Alerts ----

export function listAlerts(filter?: { geofence_id?: string; vehicle_id?: string }) {
  const params = new URLSearchParams();
  if (filter?.geofence_id) params.set("geofence_id", filter.geofence_id);
  if (filter?.vehicle_id) params.set("vehicle_id", filter.vehicle_id);
  const q = params.toString() ? `?${params}` : "";
  return request<{ alerts: AlertConfig[] }>(`/alerts${q}`).then((r) => r.alerts);
}

export function configureAlert(input: {
  geofence_id: string;
  vehicle_id?: string;
  event_type: EventType;
}) {
  return request<{ alert_id: string }>("/alerts/configure", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

// ---- Violations ----

export function listViolations(filter?: {
  vehicle_id?: string;
  geofence_id?: string;
  start_date?: string;
  end_date?: string;
  limit?: number;
}) {
  const params = new URLSearchParams();
  if (filter?.vehicle_id) params.set("vehicle_id", filter.vehicle_id);
  if (filter?.geofence_id) params.set("geofence_id", filter.geofence_id);
  if (filter?.start_date) params.set("start_date", filter.start_date);
  if (filter?.end_date) params.set("end_date", filter.end_date);
  if (filter?.limit) params.set("limit", String(filter.limit));
  const q = params.toString() ? `?${params}` : "";
  return request<{ violations: Violation[]; total_count: number }>(
    `/violations/history${q}`,
  );
}
