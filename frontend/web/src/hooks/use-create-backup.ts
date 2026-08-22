import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Backup } from "../api/types";
import { qk } from "./query-keys";

export function useCreateBackup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost<Backup>("/api/v1/backups"),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.backups }),
  });
}
