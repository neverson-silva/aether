import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { ClusterServer } from "./types";

export function useServers() {
  return useQuery({ queryKey: ["servers"], queryFn: () => apiGet<ClusterServer[]>("/api/v1/servers") });
}
