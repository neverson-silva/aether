import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Worker } from "../api/types";

export function useCreateWorker(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; command: string }) =>
      apiPost<Worker>(`/api/v1/apps/${appID}/workers`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workers", appID] }),
  });
}
