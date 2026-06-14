export type Category =
  | "delivery_zone"
  | "restricted_zone"
  | "toll_zone"
  | "customer_area";

export type EventType = "entry" | "exit" | "both";

/** [latitude, longitude] */
export type Coordinate = [number, number];

export interface Geofence {
  id: string;
  name: string;
  description: string;
  coordinates: Coordinate[];
  category: Category;
  created_at: string;
}

export interface Vehicle {
  id: string;
  vehicle_number: string;
  driver_name: string;
  vehicle_type: string;
  phone: string;
  status: string;
  created_at: string;
}

export interface AlertConfig {
  alert_id: string;
  geofence_id: string;
  geofence_name: string;
  vehicle_id?: string;
  vehicle_number?: string;
  event_type: EventType;
  status: string;
  created_at: string;
}

export interface Violation {
  id: string;
  vehicle_id: string;
  vehicle_number: string;
  geofence_id: string;
  geofence_name: string;
  event_type: "entry" | "exit";
  latitude: number;
  longitude: number;
  timestamp: string;
}

export interface CurrentGeofence {
  geofence_id: string;
  geofence_name: string;
  status?: string;
  category?: Category;
}

export interface VehicleLocation {
  vehicle_id: string;
  vehicle_number: string;
  current_location: {
    latitude: number;
    longitude: number;
    timestamp: string;
  } | null;
  current_geofences: CurrentGeofence[];
}

/** Real-time alert pushed over the WebSocket. */
export interface AlertMessage {
  event_id: string;
  event_type: "entry" | "exit";
  timestamp: string;
  vehicle: {
    vehicle_id: string;
    vehicle_number: string;
    driver_name: string;
  };
  geofence: {
    geofence_id: string;
    geofence_name: string;
    category: Category;
  };
  location: {
    latitude: number;
    longitude: number;
  };
}

export const CATEGORIES: Category[] = [
  "delivery_zone",
  "restricted_zone",
  "toll_zone",
  "customer_area",
];

export const CATEGORY_COLORS: Record<Category, string> = {
  delivery_zone: "#2563eb",
  restricted_zone: "#dc2626",
  toll_zone: "#d97706",
  customer_area: "#16a34a",
};
