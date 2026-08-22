import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteSnapshotSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/snapshots/schedules/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snap-schedules"] }),
  });
}
