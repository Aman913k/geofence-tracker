import { useEffect, useState } from "react";
import { createGeofence, listGeofences } from "../api";
import type { Category, Coordinate, Geofence } from "../types";
import { CATEGORIES } from "../types";
import MapView from "../components/MapView";
import { Button, Card, CategoryBadge, ErrorText, Field, Input, Select, Spinner } from "../components/ui";

export default function GeofencesPage() {
  const [geofences, setGeofences] = useState<Geofence[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<Category | "">("");

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState<Category>("delivery_zone");
  const [draft, setDraft] = useState<Coordinate[]>([]);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const load = async (cat: Category | "") => {
    setLoading(true);
    try {
      setGeofences(await listGeofences(cat || undefined));
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load(filter);
  }, [filter]);

  const addVertex = (lat: number, lng: number) =>
    setDraft((d) => [...d, [Number(lat.toFixed(6)), Number(lng.toFixed(6))]]);

  const submit = async () => {
    setError("");
    if (!name.trim()) return setError("Name is required");
    if (draft.length < 3) return setError("Click at least 3 points on the map to define the polygon");

    // Close the polygon: first and last coordinates must match.
    const coords: Coordinate[] = [...draft];
    const [f] = coords;
    const l = coords[coords.length - 1];
    if (f[0] !== l[0] || f[1] !== l[1]) coords.push([f[0], f[1]]);

    setSaving(true);
    try {
      await createGeofence({ name, description, category, coordinates: coords });
      setName("");
      setDescription("");
      setDraft([]);
      await load(filter);
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card title="Create Geofence">
        <div className="space-y-4">
          <Field label="Name">
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Downtown Delivery Zone" />
          </Field>
          <Field label="Description">
            <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Optional description" />
          </Field>
          <Field label="Category">
            <Select value={category} onChange={(e) => setCategory(e.target.value as Category)}>
              {CATEGORIES.map((c) => (
                <option key={c} value={c}>
                  {c.replace("_", " ")}
                </option>
              ))}
            </Select>
          </Field>

          <div>
            <div className="flex items-center justify-between mb-1">
              <span className="text-sm font-medium text-slate-600">
                Boundary — click the map to add points ({draft.length})
              </span>
              <Button variant="secondary" type="button" onClick={() => setDraft([])} disabled={!draft.length}>
                Clear
              </Button>
            </div>
            <MapView geofences={geofences} draft={draft} onPick={addVertex} className="h-[360px] w-full rounded-lg overflow-hidden border border-slate-300" />
          </div>

          <ErrorText>{error}</ErrorText>
          <Button onClick={submit} disabled={saving}>
            {saving ? "Saving…" : "Create Geofence"}
          </Button>
        </div>
      </Card>

      <Card
        title="Geofences"
        actions={
          <Select value={filter} onChange={(e) => setFilter(e.target.value as Category | "")} className="w-44">
            <option value="">All categories</option>
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {c.replace("_", " ")}
              </option>
            ))}
          </Select>
        }
      >
        {loading ? (
          <Spinner />
        ) : geofences.length === 0 ? (
          <p className="text-sm text-slate-500">No geofences yet.</p>
        ) : (
          <ul className="divide-y divide-slate-100">
            {geofences.map((g) => (
              <li key={g.id} className="py-3 flex items-start justify-between gap-3">
                <div>
                  <div className="font-medium text-slate-800">{g.name}</div>
                  {g.description && <div className="text-sm text-slate-500">{g.description}</div>}
                  <div className="text-xs text-slate-400 mt-0.5">
                    {g.coordinates.length} points · {g.id}
                  </div>
                </div>
                <CategoryBadge category={g.category} />
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
