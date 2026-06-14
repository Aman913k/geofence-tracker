import { MapContainer, TileLayer, Polygon, Marker, Popup, useMapEvents, Circle } from "react-leaflet";
import L from "leaflet";
import markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import markerIcon from "leaflet/dist/images/marker-icon.png";
import markerShadow from "leaflet/dist/images/marker-shadow.png";
import type { Coordinate, Geofence } from "../types";
import { CATEGORY_COLORS } from "../types";

// Fix Leaflet's default marker icon paths under a bundler (otherwise 404s).
L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

export interface MarkerData {
  id: string;
  label: string;
  position: Coordinate;
}

interface MapViewProps {
  geofences?: Geofence[];
  markers?: MarkerData[];
  draft?: Coordinate[];
  onPick?: (lat: number, lng: number) => void;
  center?: Coordinate;
  zoom?: number;
  className?: string;
}

function ClickHandler({ onPick }: { onPick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(e) {
      onPick(e.latlng.lat, e.latlng.lng);
    },
  });
  return null;
}

export default function MapView({
  geofences = [],
  markers = [],
  draft = [],
  onPick,
  center = [37.7799, -122.4144],
  zoom = 13,
  className = "h-[420px] w-full rounded-lg overflow-hidden border border-slate-300",
}: MapViewProps) {
  return (
    <div className={className}>
      <MapContainer center={center} zoom={zoom} className="h-full w-full" scrollWheelZoom>
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        {onPick && <ClickHandler onPick={onPick} />}

        {geofences.map((g) => (
          <Polygon
            key={g.id}
            positions={g.coordinates as Coordinate[]}
            pathOptions={{
              color: CATEGORY_COLORS[g.category],
              fillColor: CATEGORY_COLORS[g.category],
              fillOpacity: 0.2,
              weight: 2,
            }}
            eventHandlers={onPick ? { click: (e) => onPick(e.latlng.lat, e.latlng.lng) }
            : undefined}
    
          >
          {!onPick && (
            <Popup>
              <strong>{g.name}</strong>
              <br />
              {g.category}
            </Popup>
          )}
          </Polygon>
        ))}

        {draft.length > 0 && (
          <>
            <Polygon
              positions={draft}
              interactive={false}
              pathOptions={{ color: "#7c3aed", dashArray: "6", fillOpacity: 0.1 }}
            />
            {draft.map((p, i) => (
              // <Circle key={i} center={p} radius={12} pathOptions={{ color: "#7c3aed" }} />
              <Circle key={i} center={p} radius={12} interactive={false} pathOptions={{ color: "#7c3aed" }} />
            ))}
          </>
        )}

        {markers.map((m) => (
          <Marker key={m.id} position={m.position}>
            <Popup>{m.label}</Popup>
          </Marker>
        ))}
      </MapContainer>
    </div>
  );
}
