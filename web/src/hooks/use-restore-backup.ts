import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export function useRestoreBackup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/backups/${id}/restore`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.backups }),
  });
}
