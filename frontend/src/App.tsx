import { useState } from "react";
import GeofencesPage from "./pages/GeofencesPage";
import VehiclesPage from "./pages/VehiclesPage";
import TrackingPage from "./pages/TrackingPage";
import AlertsPage from "./pages/AlertsPage";
import ViolationsPage from "./pages/ViolationsPage";
import { useAlerts } from "./useAlerts";
import { AlertFeed, ConnectionPill, Toasts } from "./components/LiveAlerts";

const TABS = [
  { key: "geofences", label: "Geofences", el: <GeofencesPage /> },
  { key: "vehicles", label: "Vehicles", el: <VehiclesPage /> },
  { key: "tracking", label: "Tracking", el: <TrackingPage /> },
  { key: "alerts", label: "Alert Rules", el: <AlertsPage /> },
  { key: "history", label: "History", el: <ViolationsPage /> },
] as const;

export default function App() {
  const [tab, setTab] = useState<(typeof TABS)[number]["key"]>("geofences");
  const [feedOpen, setFeedOpen] = useState(false);
  const { alerts, state, clear } = useAlerts();

  return (
    <div className="min-h-full">
      <header className="bg-white border-b border-slate-200 sticky top-0 z-[900]">
        <div className="max-w-6xl mx-auto px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-xl">🛰️</span>
            <h1 className="text-lg font-bold text-slate-800">Geofence &amp; Vehicle Tracker</h1>
          </div>
          <div className="flex items-center gap-4">
            <ConnectionPill state={state} />
            <button
              onClick={() => setFeedOpen((o) => !o)}
              className="relative rounded-md bg-slate-100 hover:bg-slate-200 px-3 py-1.5 text-sm font-medium"
            >
              🔔 Alerts
              {alerts.length > 0 && (
                <span className="absolute -top-1 -right-1 bg-red-600 text-white text-xs rounded-full h-5 min-w-5 px-1 flex items-center justify-center">
                  {alerts.length > 99 ? "99+" : alerts.length}
                </span>
              )}
            </button>
          </div>
        </div>
        <nav className="max-w-6xl mx-auto px-4 flex gap-1 overflow-x-auto">
          {TABS.map((t) => (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`px-4 py-2.5 text-sm font-medium border-b-2 -mb-px whitespace-nowrap ${
                tab === t.key
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-slate-500 hover:text-slate-700"
              }`}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </header>

      <main className="max-w-6xl mx-auto px-4 py-6">{TABS.find((t) => t.key === tab)?.el}</main>

      {/* Slide-over live alert feed */}
      {feedOpen && (
        <>
          <div className="fixed inset-0 bg-black/20 z-[950]" onClick={() => setFeedOpen(false)} />
          <aside className="fixed top-0 right-0 h-full w-full max-w-md bg-white shadow-xl z-[960] p-5 overflow-y-auto">
            <AlertFeed alerts={alerts} onClear={clear} />
          </aside>
        </>
      )}

      <Toasts alerts={alerts} />
    </div>
  );
}
