import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { GitOpsConfig } from "./types";

export function useSyncGitOps() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<GitOpsConfig>(`/api/v1/gitops/${id}/sync`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gitops"] }),
  });
}
