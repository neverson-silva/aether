import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Domain } from "../api/types";
import { qk } from "./query-keys";

export function useGenerateFreeDomain(kind: string, id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (https?: boolean) => apiPost<Domain>(`/api/v1/${kind}/${id}/domains/generate`, { https: https ?? true }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(kind, id) }),
  });
}
