import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { SnapshotSchedule } from "./types";

export function useCreateSnapshotSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<SnapshotSchedule>) => apiPost<SnapshotSchedule>("/api/v1/snapshots/schedules", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snap-schedules"] }),
  });
}
