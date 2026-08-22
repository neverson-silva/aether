import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Snapshot } from "./types";

export function useCreateSnapshot() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { app_id: string; volume: string; name: string }) => apiPost<Snapshot>("/api/v1/snapshots", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["snapshots"] }),
  });
}
