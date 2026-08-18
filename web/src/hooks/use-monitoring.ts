import { useEffect, useState } from "react";
import type { MonitoringSnapshot } from "./types";

// Single SSE stream delivering the full monitoring snapshot (host, aether,
// user, per-resource metrics) — never one request per container.
export function useMonitoring() {
  const [snapshot, setSnapshot] = useState<MonitoringSnapshot | null>(null);
  const [connected, setConnected] = useState(false);
  useEffect(() => {
    let active = true;
    const server = localStorage.getItem("aether_server") || "";
    const es = new EventSource(server + "/api/v1/monitoring/stream", { withCredentials: true });
    es.onopen = () => active && setConnected(true);
    es.onerror = () => active && setConnected(false);
    es.addEventListener("monitoring", (ev) => {
      if (!active) return;
      try {
        setSnapshot(JSON.parse((ev as MessageEvent).data) as MonitoringSnapshot);
      } catch {
      }
    });
    return () => {
      active = false;
      es.close();
    };
  }, []);
  return { snapshot, connected };
}
