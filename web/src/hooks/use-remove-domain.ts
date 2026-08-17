import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";
import { qk } from "./query-keys";

export function useRemoveDomain(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (host: string) => apiDelete(`/api/v1/apps/${appID}/domains/${host}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(appID) }),
  });
}
