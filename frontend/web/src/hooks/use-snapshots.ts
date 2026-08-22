import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Snapshot } from "./types";

export function useSnapshots() {
  return useQuery({ queryKey: ["snapshots"], queryFn: () => apiGet<Snapshot[]>("/api/v1/snapshots") });
}
