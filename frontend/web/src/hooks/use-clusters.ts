import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Cluster } from "./types";

export function useClusters() {
  return useQuery({ queryKey: ["clusters"], queryFn: () => apiGet<Cluster[]>("/api/v1/clusters") });
}
