import { useEffect, useState } from "react";
import { apiGet, apiPost } from "../api/client";

export function usePresence(scope: string) {
  const [count, setCount] = useState(0);
  useEffect(() => {
    if (!scope) return;
    let active = true;
    apiPost("/api/v1/presence/join", { scope }).catch(() => {});
    const beat = setInterval(() => {
      apiPost("/api/v1/presence/heartbeat", { scope }).catch(() => {});
    }, 30000);
    const tick = setInterval(() => {
      apiGet<{ count: number }>("/api/v1/presence/count?scope=" + encodeURIComponent(scope))
        .then((r) => { if (active) setCount(r.count); })
        .catch(() => {});
    }, 10000);
    return () => {
      active = false;
      clearInterval(beat);
      clearInterval(tick);
      apiPost("/api/v1/presence/leave", { scope }).catch(() => {});
    };
  }, [scope]);
  return count;
}
