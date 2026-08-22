import { useEffect, useRef, useState } from "react";
import { apiGet } from "../api/client";
import { useRealtime, useRealtimeEvent } from "../components/RealtimeProvider";

export function useAppStates() {
  const [states, setStates] = useState<Record<string, string>>({});
  const { connected } = useRealtime();
  const prevConnected = useRef(connected);

  useEffect(() => {
    let active = true;
    const load = () =>
      apiGet<Record<string, string>>("/api/v1/apps/states")
        .then((s) => {
          if (active) setStates(s);
        })
        .catch(() => {});
    load();
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (prevConnected.current === false && connected === true) {
      apiGet<Record<string, string>>("/api/v1/apps/states")
        .then((s) => setStates(s))
        .catch(() => {});
    }
    prevConnected.current = connected;
  }, [connected]);

  useRealtimeEvent((ev, replay) => {
    if (ev.type === "app.state") {
      const p = ev.payload as { app_id?: string; state?: string } | undefined;
      if (!p?.app_id || !p?.state) return;
      setStates((prev) => ({ ...prev, [p.app_id as string]: p.state as string }));
      return;
    }
    if (replay) return;
    if (ev.type === "deploy.failed" || ev.type === "deploy.ready" || ev.type === "deploy.rolled_back" || ev.type === "deploy.cancelled") {
      apiGet<Record<string, string>>("/api/v1/apps/states")
        .then((s) => setStates(s))
        .catch(() => {});
    }
  });

  return { data: states };
}