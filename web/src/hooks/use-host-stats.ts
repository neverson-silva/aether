import { useEffect, useState } from "react";
import type { HostStats } from "./types";

export function useHostStats() {
  const [stats, setStats] = useState<HostStats | null>(null);
  const [history, setHistory] = useState<{ cpu: number; mem: number }[]>([]);
  useEffect(() => {
    let active = true;
    const server = localStorage.getItem("aether_server") || "";
    const es = new EventSource(server + "/api/v1/host/stats/stream", { withCredentials: true });
    es.addEventListener("stats", (ev) => {
      if (!active) return;
      try {
        const s = JSON.parse((ev as MessageEvent).data) as HostStats;
        setStats(s);
        setHistory((prev) => [...prev.slice(-59), { cpu: s.cpu_percent, mem: s.mem_percent }]);
      } catch { /* ignore */ }
    });
    return () => { active = false; es.close(); };
  }, []);
  return { stats, history };
}
