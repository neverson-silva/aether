import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPut } from "../api/client";
import type { AppPolicy } from "./types";

export function useSavePolicy(appId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (p: AppPolicy) => apiPut<AppPolicy>(`/api/v1/apps/${appId}/policy`, p),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["policy", appId] }),
  });
}
