# 🛰️ Geofencing & Real-time Alert System

Define virtual boundaries (geofences), track vehicles in real time, and get
**instant WebSocket alerts** when a vehicle enters or exits a zone.

- **Backend** — Go (gorilla/mux + gorilla/websocket), PostgreSQL + **PostGIS**
- **Frontend** — React + Vite + TypeScript + Tailwind + **Leaflet**
- **Real-time** — WebSocket broadcast at `/ws/alerts`
- **Infra** — Docker Compose (db + backend + frontend)

- **Loom Link** — https://www.loom.com/share/659464beca284e08ba02fa4474de57a2

## Run it

```bash
docker compose up --build
# frontend → http://localhost:3000
# backend  → http://localhost:8080
```

See **[SETUP.md](./SETUP.md)** for prerequisites, local (non-Docker) setup, the
full API reference with curl examples, the frontend guide, and the architecture
overview.

## Endpoints (all responses carry `time_ns`)

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/geofences` | Create a polygonal geofence |
| GET | `/geofences` | List geofences (`?category=`) |
| POST | `/vehicles` | Register a vehicle |
| GET | `/vehicles` | List vehicles |
| POST | `/vehicles/location` | Update location → detect entry/exit → fire alerts |
| GET | `/vehicles/location/{vehicle_id}` | Current location + geofence status |
| POST | `/alerts/configure` | Configure an alert rule |
| GET | `/alerts` | List alert rules (`?geofence_id=` / `?vehicle_id=`) |
| GET | `/violations/history` | Entry/exit history (filters + pagination) |
| WS | `/ws/alerts` | Live alert stream |
