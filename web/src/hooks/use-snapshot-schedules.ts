import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { SnapshotSchedule } from "./types";

export function useSnapshotSchedules() {
  return useQuery({ queryKey: ["snap-schedules"], queryFn: () => apiGet<SnapshotSchedule[]>("/api/v1/snapshots/schedules") });
}
