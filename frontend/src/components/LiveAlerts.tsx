import { useEffect, useRef, useState } from "react";
import type { AlertMessage } from "../types";
import type { ConnectionState } from "../useAlerts";
import { Badge, CategoryBadge } from "./ui";

export function ConnectionPill({ state }: { state: ConnectionState }) {
  const map: Record<ConnectionState, { color: string; label: string }> = {
    open: { color: "bg-green-500", label: "Live" },
    connecting: { color: "bg-amber-400 animate-pulse", label: "Connecting" },
    closed: { color: "bg-red-500", label: "Disconnected" },
  };
  const { color, label } = map[state];
  return (
    <span className="inline-flex items-center gap-2 text-sm text-slate-600">
      <span className={`h-2.5 w-2.5 rounded-full ${color}`} />
      {label}
    </span>
  );
}

interface ToastItem {
  alert: AlertMessage;
  shownAt: number;
}

/**
 * Toasts surfaces the newest incoming alerts as transient cards in the bottom-right
 * corner, auto-dismissing after a few seconds. It tracks already-shown alerts by
 * event_id so re-renders never duplicate a toast.
 */
export function Toasts({ alerts }: { alerts: AlertMessage[] }) {
  const [visible, setVisible] = useState<ToastItem[]>([]);
  const seen = useRef<Set<string>>(new Set());

  useEffect(() => {
    const fresh = alerts.filter((a) => !seen.current.has(a.event_id));
    if (fresh.length === 0) return;
    fresh.forEach((a) => seen.current.add(a.event_id));
    setVisible((prev) => [...fresh.map((alert) => ({ alert, shownAt: Date.now() })), ...prev].slice(0, 5));
  }, [alerts]);

  useEffect(() => {
    if (visible.length === 0) return;
    const timer = window.setInterval(() => {
      const cutoff = Date.now() - 8000;
      setVisible((prev) => prev.filter((t) => t.shownAt > cutoff));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [visible.length]);

  return (
    <div className="fixed bottom-4 right-4 z-[1000] flex flex-col gap-2 w-80">
      {visible.map(({ alert }) => (
        <div
          key={alert.event_id}
          className={`rounded-lg shadow-lg border bg-white p-3 animate-[fadeIn_0.2s_ease-out] ${
            alert.geofence.category === "restricted_zone" ? "border-red-300" : "border-slate-200"
          }`}
        >
          <div className="flex items-center justify-between mb-1">
            <span className="font-semibold text-slate-800">
              {alert.event_type === "entry" ? "🚧 Entered" : "↩️ Exited"} zone
            </span>
            <Badge color={alert.event_type === "exit" ? "amber" : "green"}>{alert.event_type}</Badge>
          </div>
          <div className="text-sm text-slate-600">
            <strong>{alert.vehicle.vehicle_number}</strong> ({alert.vehicle.driver_name})
          </div>
          <div className="text-sm text-slate-600 flex items-center gap-2 mt-0.5">
            {alert.geofence.geofence_name} <CategoryBadge category={alert.geofence.category} />
          </div>
        </div>
      ))}
    </div>
  );
}

/** AlertFeed renders the full session history of received alerts. */
export function AlertFeed({ alerts, onClear }: { alerts: AlertMessage[]; onClear: () => void }) {
  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-lg font-semibold text-slate-800">Live Alert Feed</h2>
        {alerts.length > 0 && (
          <button onClick={onClear} className="text-sm text-slate-500 hover:text-slate-700">
            Clear
          </button>
        )}
      </div>
      {alerts.length === 0 ? (
        <p className="text-sm text-slate-500">
          No alerts yet. Configure an alert rule, then move a vehicle in/out of its geofence.
        </p>
      ) : (
        <ul className="space-y-2 max-h-[70vh] overflow-y-auto pr-1">
          {alerts.map((a) => (
            <li key={a.event_id} className="rounded-lg border border-slate-200 p-3 text-sm">
              <div className="flex items-center justify-between">
                <span className="font-medium">{a.vehicle.vehicle_number}</span>
                <Badge color={a.event_type === "exit" ? "amber" : "green"}>{a.event_type}</Badge>
              </div>
              <div className="text-slate-600 mt-0.5 flex items-center gap-2">
                {a.geofence.geofence_name} <CategoryBadge category={a.geofence.category} />
              </div>
              <div className="text-xs text-slate-400 mt-1">
                {new Date(a.timestamp).toLocaleString()} · {a.location.latitude.toFixed(4)},{" "}
                {a.location.longitude.toFixed(4)}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
