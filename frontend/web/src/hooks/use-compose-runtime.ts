import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Stats, TimelineEvent } from "../api/types";

export function useComposeStats(id: string) {
  return useQuery({ queryKey: ["compose", id, "stats"], queryFn: () => apiGet<Stats>(`/api/v1/compose/${id}/stats`), enabled: !!id });
}

export function useComposeTimeline(id: string) {
  return useQuery({ queryKey: ["compose", id, "timeline"], queryFn: () => apiGet<TimelineEvent[]>(`/api/v1/compose/${id}/timeline`), enabled: !!id });
}

export function useComposeEnv(id: string) {
  return useQuery({ queryKey: ["compose", id, "env"], queryFn: () => apiGet<{ env: unknown[] }>(`/api/v1/compose/${id}/env`), enabled: !!id });
}

export function useComposeDeployments(id: string) {
  return useQuery({ queryKey: ["compose", id, "deployments"], queryFn: () => apiGet<unknown[]>(`/api/v1/apps/${id}/deployments`), enabled: !!id });
}
