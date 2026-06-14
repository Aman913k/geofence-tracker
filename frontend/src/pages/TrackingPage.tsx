import { useEffect, useMemo, useState } from "react";
import { getVehicleLocation, listGeofences, listVehicles, updateLocation } from "../api";
import type { CurrentGeofence, Geofence, Vehicle } from "../types";
import MapView, { type MarkerData } from "../components/MapView";
import { Badge, Button, Card, ErrorText, Field, Input, Select } from "../components/ui";

function nowISO() {
  return new Date().toISOString().slice(0, 19) + "Z";
}

export default function TrackingPage() {
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [geofences, setGeofences] = useState<Geofence[]>([]);
  const [vehicleId, setVehicleId] = useState("");
  const [lat, setLat] = useState("37.7799");
  const [lng, setLng] = useState("-122.4144");
  const [timestamp, setTimestamp] = useState(nowISO());
  const [current, setCurrent] = useState<CurrentGeofence[] | null>(null);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    listVehicles().then(setVehicles).catch((e) => setError(String(e)));
    listGeofences().then(setGeofences).catch((e) => setError(String(e)));
  }, []);

  // Load the selected vehicle's last known location and geofence status.
  useEffect(() => {
    if (!vehicleId) {
      setCurrent(null);
      return;
    }
    getVehicleLocation(vehicleId)
      .then((loc) => {
        setCurrent(loc.current_geofences);
        if (loc.current_location) {
          setLat(String(loc.current_location.latitude));
          setLng(String(loc.current_location.longitude));
        }
      })
      .catch(() => setCurrent(null));
  }, [vehicleId]);

  const markers: MarkerData[] = useMemo(() => {
    const la = parseFloat(lat);
    const ln = parseFloat(lng);
    if (Number.isNaN(la) || Number.isNaN(ln)) return [];
    const v = vehicles.find((x) => x.id === vehicleId);
    return [{ id: "picked", label: v ? v.vehicle_number : "Selected location", position: [la, ln] }];
  }, [lat, lng, vehicleId, vehicles]);

  const submit = async () => {
    setError("");
    if (!vehicleId) return setError("Select a vehicle");
    const la = parseFloat(lat);
    const ln = parseFloat(lng);
    if (Number.isNaN(la) || Number.isNaN(ln)) return setError("Latitude/longitude must be numbers");

    setSaving(true);
    try {
      const res = await updateLocation({
        vehicle_id: vehicleId,
        latitude: la,
        longitude: ln,
        timestamp: timestamp || nowISO(),
      });
      setCurrent(res.current_geofences);
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card title="Update Vehicle Location">
        <div className="space-y-4">
          <Field label="Vehicle">
            <Select value={vehicleId} onChange={(e) => setVehicleId(e.target.value)}>
              <option value="">Select a vehicle…</option>
              {vehicles.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.vehicle_number} — {v.driver_name}
                </option>
              ))}
            </Select>
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Latitude">
              <Input value={lat} onChange={(e) => setLat(e.target.value)} />
            </Field>
            <Field label="Longitude">
              <Input value={lng} onChange={(e) => setLng(e.target.value)} />
            </Field>
          </div>
          <Field label="Timestamp (ISO 8601)">
            <div className="flex gap-2">
              <Input value={timestamp} onChange={(e) => setTimestamp(e.target.value)} />
              <Button variant="secondary" type="button" onClick={() => setTimestamp(nowISO())}>
                Now
              </Button>
            </div>
          </Field>
          <ErrorText>{error}</ErrorText>
          <Button onClick={submit} disabled={saving}>
            {saving ? "Updating…" : "Update Location"}
          </Button>

          {current && (
            <div className="mt-2">
              <div className="text-sm font-medium text-slate-600 mb-1">Currently inside:</div>
              {current.length === 0 ? (
                <Badge>Outside all geofences</Badge>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {current.map((g) => (
                    <Badge key={g.geofence_id} color="blue">
                      {g.geofence_name}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </Card>

      <Card title="Map — click to set location">
        <MapView
          geofences={geofences}
          markers={markers}
          center={markers[0]?.position ?? [37.7799, -122.4144]}
          onPick={(la, ln) => {
            setLat(String(Number(la.toFixed(6))));
            setLng(String(Number(ln.toFixed(6))));
          }}
        />
        <p className="text-xs text-slate-400 mt-2">
          Click anywhere on the map to drop the vehicle there, then press “Update Location”.
        </p>
      </Card>
    </div>
  );
}
