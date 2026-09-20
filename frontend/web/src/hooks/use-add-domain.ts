import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export function useAddDomain(kind: string, id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { host: string; https: boolean; container_port?: number; compose_service_name?: string; path?: string; internal_path?: string; strip_path?: boolean }) =>
      apiPost(`/api/v1/${kind}/${id}/domains`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(kind, id) }),
  });
}
