import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

type HostEvent = { ts: string; type: string; title?: string; detail?: string };

export function useHostEvents() {
  return useQuery({
    queryKey: ["host-events"],
    queryFn: () => apiGet<{ events: HostEvent[] }>("/api/v1/host/events").then((r) => r.events),
  });
}
