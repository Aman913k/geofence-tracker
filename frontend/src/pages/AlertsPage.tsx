import { useEffect, useState } from "react";
import { configureAlert, listAlerts, listGeofences, listVehicles } from "../api";
import type { AlertConfig, EventType, Geofence, Vehicle } from "../types";
import { Badge, Button, Card, ErrorText, Field, Select, Spinner } from "../components/ui";

const EVENT_TYPES: EventType[] = ["entry", "exit", "both"];

export default function AlertsPage() {
  const [alerts, setAlerts] = useState<AlertConfig[]>([]);
  const [geofences, setGeofences] = useState<Geofence[]>([]);
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);

  const [geofenceId, setGeofenceId] = useState("");
  const [vehicleId, setVehicleId] = useState("");
  const [eventType, setEventType] = useState<EventType>("entry");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const [a, g, v] = await Promise.all([listAlerts(), listGeofences(), listVehicles()]);
      setAlerts(a);
      setGeofences(g);
      setVehicles(v);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const submit = async () => {
    setError("");
    if (!geofenceId) return setError("Select a geofence");
    setSaving(true);
    try {
      await configureAlert({ geofence_id: geofenceId, vehicle_id: vehicleId || undefined, event_type: eventType });
      await load();
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card title="Configure Alert Rule">
        <div className="space-y-4">
          <Field label="Geofence">
            <Select value={geofenceId} onChange={(e) => setGeofenceId(e.target.value)}>
              <option value="">Select a geofence…</option>
              {geofences.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name} ({g.category.replace("_", " ")})
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Vehicle (optional — applies to all if empty)">
            <Select value={vehicleId} onChange={(e) => setVehicleId(e.target.value)}>
              <option value="">All vehicles</option>
              {vehicles.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.vehicle_number} — {v.driver_name}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Event Type">
            <Select value={eventType} onChange={(e) => setEventType(e.target.value as EventType)}>
              {EVENT_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </Select>
          </Field>
          <ErrorText>{error}</ErrorText>
          <Button onClick={submit} disabled={saving}>
            {saving ? "Saving…" : "Configure Alert"}
          </Button>
        </div>
      </Card>

      <Card title={`Alert Rules (${alerts.length})`}>
        {loading ? (
          <Spinner />
        ) : alerts.length === 0 ? (
          <p className="text-sm text-slate-500">No alert rules configured yet.</p>
        ) : (
          <ul className="divide-y divide-slate-100">
            {alerts.map((a) => (
              <li key={a.alert_id} className="py-3 flex items-center justify-between gap-3">
                <div>
                  <div className="font-medium text-slate-800">{a.geofence_name}</div>
                  <div className="text-sm text-slate-500">
                    {a.vehicle_number ? a.vehicle_number : "All vehicles"}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge color={a.event_type === "exit" ? "amber" : a.event_type === "both" ? "blue" : "green"}>
                    {a.event_type}
                  </Badge>
                  <Badge>{a.status}</Badge>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
