import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPatch } from "../api/client";

export function useUpdateApp(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name?: string; image_retention?: number; port?: number; resources?: { cpus?: string; mem_mb?: number } }) => apiPatch(`/api/v1/apps/${appID}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["app", appID] }),
  });
}
