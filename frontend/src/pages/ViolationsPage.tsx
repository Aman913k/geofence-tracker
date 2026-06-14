import { useEffect, useState } from "react";
import { listGeofences, listVehicles, listViolations } from "../api";
import type { Geofence, Vehicle, Violation } from "../types";
import { Badge, Button, Card, Field, Select, Spinner } from "../components/ui";

export default function ViolationsPage() {
  const [violations, setViolations] = useState<Violation[]>([]);
  const [total, setTotal] = useState(0);
  const [geofences, setGeofences] = useState<Geofence[]>([]);
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);

  const [vehicleId, setVehicleId] = useState("");
  const [geofenceId, setGeofenceId] = useState("");
  const [limit, setLimit] = useState(50);

  const load = async () => {
    setLoading(true);
    try {
      const res = await listViolations({
        vehicle_id: vehicleId || undefined,
        geofence_id: geofenceId || undefined,
        limit,
      });
      setViolations(res.violations);
      setTotal(res.total_count);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    listGeofences().then(setGeofences);
    listVehicles().then(setVehicles);
  }, []);

  useEffect(() => {
    load();
  }, [vehicleId, geofenceId, limit]);

  return (
    <Card title="Violation History" actions={<Badge color="blue">{total} total events</Badge>}>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-4">
        <Field label="Vehicle">
          <Select value={vehicleId} onChange={(e) => setVehicleId(e.target.value)}>
            <option value="">All vehicles</option>
            {vehicles.map((v) => (
              <option key={v.id} value={v.id}>
                {v.vehicle_number}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Geofence">
          <Select value={geofenceId} onChange={(e) => setGeofenceId(e.target.value)}>
            <option value="">All geofences</option>
            {geofences.map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Limit">
          <Select value={limit} onChange={(e) => setLimit(Number(e.target.value))}>
            {[20, 50, 100, 200, 500].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </Select>
        </Field>
      </div>

      {loading ? (
        <Spinner />
      ) : violations.length === 0 ? (
        <p className="text-sm text-slate-500">No violation events recorded.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-slate-500 border-b border-slate-200">
                <th className="py-2 pr-4">Time</th>
                <th className="py-2 pr-4">Vehicle</th>
                <th className="py-2 pr-4">Geofence</th>
                <th className="py-2 pr-4">Event</th>
                <th className="py-2 pr-4">Location</th>
              </tr>
            </thead>
            <tbody>
              {violations.map((v) => (
                <tr key={v.id} className="border-b border-slate-100">
                  <td className="py-2 pr-4 whitespace-nowrap">{new Date(v.timestamp).toLocaleString()}</td>
                  <td className="py-2 pr-4">{v.vehicle_number}</td>
                  <td className="py-2 pr-4">{v.geofence_name}</td>
                  <td className="py-2 pr-4">
                    <Badge color={v.event_type === "exit" ? "amber" : "green"}>{v.event_type}</Badge>
                  </td>
                  <td className="py-2 pr-4 text-slate-500">
                    {v.latitude.toFixed(4)}, {v.longitude.toFixed(4)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="mt-3 flex items-center justify-between text-xs text-slate-400">
            <span>
              Showing {violations.length} of {total}
            </span>
            <Button variant="secondary" onClick={load}>
              Refresh
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
}
