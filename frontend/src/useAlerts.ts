import { useEffect, useRef, useState } from "react";
import type { AlertMessage } from "./types";

const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/alerts";

export type ConnectionState = "connecting" | "open" | "closed";

/**
 * useAlerts subscribes to the backend WebSocket alert stream. It auto-reconnects
 * with backoff and accumulates received alerts (most recent first).
 */
export function useAlerts(max = 100) {
  const [alerts, setAlerts] = useState<AlertMessage[]>([]);
  const [state, setState] = useState<ConnectionState>("connecting");
  const retryRef = useRef(0);
  const closedByUnmount = useRef(false);

  useEffect(() => {
    closedByUnmount.current = false;
    let ws: WebSocket | null = null;
    let reconnectTimer: number | undefined;

    const connect = () => {
      setState("connecting");
      ws = new WebSocket(WS_URL);

      ws.onopen = () => {
        retryRef.current = 0;
        setState("open");
      };

      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data) as AlertMessage;
          setAlerts((prev) => [msg, ...prev].slice(0, max));
        } catch {
          // ignore non-JSON frames
        }
      };

      ws.onclose = () => {
        setState("closed");
        if (closedByUnmount.current) return;
        const delay = Math.min(1000 * 2 ** retryRef.current, 10000);
        retryRef.current += 1;
        reconnectTimer = window.setTimeout(connect, delay);
      };

      ws.onerror = () => ws?.close();
    };

    connect();

    return () => {
      closedByUnmount.current = true;
      if (reconnectTimer) window.clearTimeout(reconnectTimer);
      ws?.close();
    };
  }, [max]);

  const clear = () => setAlerts([]);

  return { alerts, state, clear };
}
