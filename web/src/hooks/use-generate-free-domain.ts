import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Domain } from "../api/types";
import { qk } from "./query-keys";

export function useGenerateFreeDomain(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (https?: boolean) => apiPost<Domain>(`/api/v1/apps/${appID}/domains/generate`, { https: https ?? true }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(appID) }),
  });
}
