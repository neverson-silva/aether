import { useEffect, useState } from "react";
import { apiPost, getOrgId, getServer } from "../api/client";
import type { MonitoringSnapshot } from "./types";

export function useMonitoring() {
  const [snapshot, setSnapshot] = useState<MonitoringSnapshot | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    const wait = (milliseconds: number) => new Promise<void>((resolve) => window.setTimeout(resolve, milliseconds));

    const consume = async () => {
      while (active) {
        try {
          const response = await fetch(`${getServer()}/api/v1/monitoring/stream`, {
            credentials: "include",
            headers: { Accept: "text/event-stream", "X-Aether-Org": getOrgId() },
            signal: controller.signal,
          });
          if (response.status === 401) {
            await apiPost("/api/v1/auth/refresh");
            continue;
          }
          if (!response.ok) throw new Error(`Monitoring stream failed with HTTP ${response.status}`);
          if (!response.body) throw new Error("Monitoring stream returned no body");
          const contentType = response.headers.get("content-type") || "";
          if (!contentType.toLowerCase().includes("text/event-stream")) throw new Error("Monitoring stream returned an invalid content type");
          active && setConnected(true);
          active && setError(null);
          const reader = response.body.getReader();
          const decoder = new TextDecoder();
          let buffer = "";
          while (active) {
            const { value, done } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            const events = buffer.split("\n\n");
            buffer = events.pop() || "";
            for (const event of events) {
              const data = event.split("\n").find((line) => line.startsWith("data: "))?.slice(6);
              if (!data) continue;
              try {
                setSnapshot(JSON.parse(data) as MonitoringSnapshot);
              } catch {
                setError("Monitoring returned an invalid snapshot");
              }
            }
          }
          if (active) await wait(2000);
        } catch (streamError) {
          if (!active || (streamError instanceof DOMException && streamError.name === "AbortError")) return;
          setConnected(false);
          setError(streamError instanceof Error ? streamError.message : "Monitoring stream unavailable");
          await wait(2000);
        } finally {
          if (active) setConnected(false);
        }
      }
    };

    void consume();
    return () => {
      active = false;
      controller.abort();
    };
  }, []);
  return { snapshot, connected, error };
}
