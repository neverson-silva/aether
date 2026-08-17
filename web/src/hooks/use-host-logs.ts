import { useEffect, useState } from "react";

export function useHostLogs(follow: boolean) {
  const [lines, setLines] = useState<{ line: string }[]>([]);
  useEffect(() => {
    if (!follow) return;
    let active = true;
    const server = localStorage.getItem("aether_server") || "";
    const es = new EventSource(server + "/api/v1/host/logs?follow=1", { withCredentials: true });
    es.addEventListener("log", (ev) => {
      if (!active) return;
      try {
        const l = JSON.parse((ev as MessageEvent).data) as { line: string };
        setLines((prev) => [...prev, l].slice(-400));
      } catch { /* ignore */ }
    });
    return () => { active = false; es.close(); };
  }, [follow]);
  return lines;
}
