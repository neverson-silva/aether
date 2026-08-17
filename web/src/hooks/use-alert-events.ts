import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { AlertEvent } from "./types";

export function useAlertEvents(limit = 30) {
  return useQuery({ queryKey: ["alert-events", limit], queryFn: () => apiGet<AlertEvent[]>(`/api/v1/alerts/events?limit=${limit}`) });
}
