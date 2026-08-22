import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteGitOps() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/gitops/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["gitops"] }),
  });
}
