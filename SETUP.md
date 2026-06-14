# SETUP — Geofencing & Real-time Alert System

A full-stack geofencing and vehicle-tracking system:

- **Backend** — Go REST API + WebSocket alert stream, PostgreSQL + **PostGIS** for geospatial containment.
- **Frontend** — React (Vite + TypeScript + Tailwind) with an interactive Leaflet map.
- **Real-time** — WebSocket (`/ws/alerts`) broadcasts entry/exit alerts to all connected clients.

---

## 1. Prerequisites

| Tool | Version | Needed for |
|------|---------|-----------|
| Docker + Docker Compose | recent | Running the whole stack (recommended) |
| Go | 1.24+ | Running the backend without Docker |
| Node.js | 20+ | Running the frontend without Docker |

The only hard dependency for the recommended path is **Docker**.

---

## 2. Quick start with Docker Compose (recommended)

From the project root:

```bash
docker compose up --build
```

This starts three services:

| Service  | URL                                   | Notes |
|----------|---------------------------------------|-------|
| frontend | http://localhost:3000                 | React app (nginx) |
| backend  | http://localhost:8080                 | REST API + WebSocket |
| db       | localhost:**5432** → container 5432   | PostGIS 16 |

> The database is published on host port **5432** 
> with a local Postgres. The backend always reaches it internally at `db:5432`.

Open **http://localhost:3000** and you're ready.

Shut down (and wipe the DB volume):

```bash
docker compose down -v
```

---

## 3. Running locally without Docker

### Backend

```bash
# Start a PostGIS instance (or point DATABASE_URL at your own)
docker run -d --name gf-db -p 5432:5432 \
  -e POSTGRES_USER=geofence -e POSTGRES_PASSWORD=geofence -e POSTGRES_DB=geofence \
  postgis/postgis:16-3.4

cd backend
export DATABASE_URL="postgres://geofence:geofence@localhost:5432/geofence?sslmode=disable"
export LISTEN_ADDR=":8080"
go run .
```

The schema (tables, PostGIS extension, indexes) is created automatically on startup.

### Frontend

```bash
cd frontend
npm install
# Point the app at your backend (defaults shown):
export VITE_API_BASE_URL="http://localhost:8080"
export VITE_WS_URL="ws://localhost:8080/ws/alerts"
npm run dev          # dev server on http://localhost:5173
```

---

## 4. Configuration

All configuration is via environment variables (see `.env.example`).

| Variable | Service | Default | Description |
|----------|---------|---------|-------------|
| `DATABASE_URL` | backend | `postgres://geofence:geofence@db:5432/geofence?sslmode=disable` | Postgres/PostGIS DSN |
| `LISTEN_ADDR` | backend | `:8080` | HTTP listen address |
| `VITE_API_BASE_URL` | frontend (build-time) | `http://localhost:8080` | REST base URL baked into the bundle |
| `VITE_WS_URL` | frontend (build-time) | `ws://localhost:8080/ws/alerts` | WebSocket URL baked into the bundle |

---

## 5. API testing guide (curl)

> Every response includes `"time_ns"` — the request handler execution time in nanoseconds.

```bash
B=http://localhost:8080

# 1) Create a (restricted) geofence — note coordinates are [lat, lng], closed ring
GEO=$(curl -s -X POST $B/geofences -H 'Content-Type: application/json' -d '{
  "name":"Restricted Yard",
  "description":"No entry",
  "category":"restricted_zone",
  "coordinates":[[37.7749,-122.4194],[37.7849,-122.4194],[37.7849,-122.4094],[37.7749,-122.4094],[37.7749,-122.4194]]
}')
echo "$GEO"
GID=$(echo "$GEO" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

# 2) List geofences (optional ?category= filter)
curl -s "$B/geofences?category=restricted_zone"

# 3) Register a vehicle
VEH=$(curl -s -X POST $B/vehicles -H 'Content-Type: application/json' -d '{
  "vehicle_number":"KA-01-AB-1234","driver_name":"John Doe","vehicle_type":"truck","phone":"+1234567890"
}')
echo "$VEH"
VID=$(echo "$VEH" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

# 4) List vehicles
curl -s $B/vehicles

# 5) Configure an alert rule (entry + exit) for this geofence + vehicle
curl -s -X POST $B/alerts/configure -H 'Content-Type: application/json' \
  -d "{\"geofence_id\":\"$GID\",\"vehicle_id\":\"$VID\",\"event_type\":\"both\"}"

# 6) List alert rules (optional ?geofence_id= / ?vehicle_id= filters)
curl -s $B/alerts

# 7) Move the vehicle INSIDE the geofence -> fires an "entry" alert over WebSocket
curl -s -X POST $B/vehicles/location -H 'Content-Type: application/json' -d "{
  \"vehicle_id\":\"$VID\",\"latitude\":37.7799,\"longitude\":-122.4144,\"timestamp\":\"2025-01-15T10:35:00Z\"
}"

# 8) Move it OUTSIDE -> fires an "exit" alert
curl -s -X POST $B/vehicles/location -H 'Content-Type: application/json' -d "{
  \"vehicle_id\":\"$VID\",\"latitude\":37.9000,\"longitude\":-122.9000,\"timestamp\":\"2025-01-15T10:40:00Z\"
}"

# 9) Current location + geofence status for a vehicle
curl -s $B/vehicles/location/$VID

# 10) Violation history (filters: vehicle_id, geofence_id, start_date, end_date, limit)
curl -s "$B/violations/history?vehicle_id=$VID&limit=100"
```

### Watching alerts over WebSocket

```bash
# Using websocat (https://github.com/vi/websocat)
websocat ws://localhost:8080/ws/alerts
```

Then run steps 7/8 above in another terminal — JSON alert messages stream in live.

---

## 6. Frontend usage guide

Open **http://localhost:3000**. The header shows the live WebSocket status
(`Live` / `Connecting` / `Disconnected`) and a 🔔 button that opens the alert feed.

1. **Geofences** — fill in name/description/category, then **click points on the map**
   to draw the boundary (the ring is auto-closed on save). Filter the list by category.
2. **Vehicles** — register vehicles; the list shows driver/type/phone/status.
3. **Tracking** — pick a vehicle, **click the map** (or type lat/lng) to set its position,
   then **Update Location**. The geofences it's currently inside are shown immediately.
4. **Alert Rules** — choose a geofence, an optional vehicle (blank = all vehicles), and
   an event type (`entry` / `exit` / `both`).
5. **History** — browse recorded entry/exit events, filter by vehicle/geofence, page via limit.
6. **Live alerts** — when a tracked move crosses a geofence with a matching rule, a toast
   pops up bottom-right and the event is added to the 🔔 feed in real time.

---

## 7. Architecture overview

```
┌─────────────┐   REST (JSON, time_ns)    ┌──────────────────────┐   SQL    ┌──────────────┐
│  React SPA  │ ────────────────────────▶ │   Go API (gorilla)   │ ───────▶ │  PostGIS DB  │
│  (Leaflet)  │ ◀─────────────────────────│  handlers/db/ws      │ ◀─────── │  ST_Contains │
└─────────────┘   WebSocket /ws/alerts     └──────────────────────┘          └──────────────┘
        ▲                                            │
        └──────────── live entry/exit alerts ────────┘
```

- **Geospatial containment** uses PostGIS `ST_Contains` against `geometry(Polygon, 4326)`
  columns with a GiST index — geofence coordinates are `[lat, lng]` and stored as `lng lat`.
- **Entry/exit detection**: on each location update the backend computes the set of
  containing geofences, diffs it against the vehicle's last stored inside-set, records each
  transition in `violations`, and (asynchronously, off the HTTP path) broadcasts an alert for
  every transition that matches an active alert rule.
- **Real-time delivery**: a `Hub` goroutine fans out broadcasts to all connected WebSocket
  clients; slow clients are dropped rather than blocking the hub.

### Backend layout

```
backend/
├── main.go                     # router + graceful shutdown
└── internal/
    ├── models/                 # domain types + validation
    ├── db/                     # pgx pool, schema migration, queries (PostGIS)
    ├── handlers/               # HTTP handlers (one file per resource) + time_ns
    ├── ws/                     # WebSocket hub
    └── middleware/             # request timing (time_ns) + CORS
```

---

## 8. Deployment notes

- **Backend image**: `docker build -t <user>/geofence-backend ./backend && docker push <user>/geofence-backend`.
  Runs on any Docker host; set `DATABASE_URL` to a managed PostGIS instance.
- **Frontend**: `cd frontend && npm run build` produces static files in `dist/` —
  deploy to Vercel / Netlify / Cloudflare Pages. Set `VITE_API_BASE_URL` and
  `VITE_WS_URL` (use `wss://` behind TLS) at build time to point at your deployed backend.
