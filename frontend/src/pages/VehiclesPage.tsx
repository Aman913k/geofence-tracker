import { useEffect, useState } from "react";
import { createVehicle, listVehicles } from "../api";
import type { Vehicle } from "../types";
import { Badge, Button, Card, ErrorText, Field, Input, Spinner } from "../components/ui";

export default function VehiclesPage() {
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({ vehicle_number: "", driver_name: "", vehicle_type: "truck", phone: "" });
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      setVehicles(await listVehicles());
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const submit = async () => {
    setError("");
    if (!form.vehicle_number || !form.driver_name || !form.vehicle_type || !form.phone) {
      return setError("All fields are required");
    }
    setSaving(true);
    try {
      await createVehicle(form);
      setForm({ vehicle_number: "", driver_name: "", vehicle_type: "truck", phone: "" });
      await load();
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card title="Register Vehicle">
        <div className="space-y-4">
          <Field label="Vehicle Number">
            <Input value={form.vehicle_number} onChange={set("vehicle_number")} placeholder="KA-01-AB-1234" />
          </Field>
          <Field label="Driver Name">
            <Input value={form.driver_name} onChange={set("driver_name")} placeholder="John Doe" />
          </Field>
          <Field label="Vehicle Type">
            <Input value={form.vehicle_type} onChange={set("vehicle_type")} placeholder="truck / car / van" />
          </Field>
          <Field label="Phone">
            <Input value={form.phone} onChange={set("phone")} placeholder="+1234567890" />
          </Field>
          <ErrorText>{error}</ErrorText>
          <Button onClick={submit} disabled={saving}>
            {saving ? "Saving…" : "Register Vehicle"}
          </Button>
        </div>
      </Card>

      <Card title={`Vehicles (${vehicles.length})`}>
        {loading ? (
          <Spinner />
        ) : vehicles.length === 0 ? (
          <p className="text-sm text-slate-500">No vehicles registered yet.</p>
        ) : (
          <ul className="divide-y divide-slate-100">
            {vehicles.map((v) => (
              <li key={v.id} className="py-3">
                <div className="flex items-center justify-between">
                  <div className="font-medium text-slate-800">{v.vehicle_number}</div>
                  <Badge color="green">{v.status}</Badge>
                </div>
                <div className="text-sm text-slate-500">
                  {v.driver_name} · {v.vehicle_type} · {v.phone}
                </div>
                <div className="text-xs text-slate-400 mt-0.5">{v.id}</div>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
