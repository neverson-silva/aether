import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { GitOpsConfig } from "./types";

export function useCreateGitOps() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; repo_url: string; branch?: string; path?: string; apply_mode?: string }) =>
      apiPost<GitOpsConfig>("/api/v1/gitops", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gitops"] }),
  });
}
