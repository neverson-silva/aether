import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Preview } from "../api/types";

export function useCreatePreview(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (branch: string) => apiPost<Preview>(`/api/v1/apps/${appID}/previews`, { branch }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["previews", appID] }),
  });
}
