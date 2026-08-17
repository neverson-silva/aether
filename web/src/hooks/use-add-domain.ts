import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export function useAddDomain(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { host: string; https: boolean; container_port?: number; path?: string; internal_path?: string; strip_path?: boolean }) =>
      apiPost(`/api/v1/apps/${appID}/domains`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(appID) }),
  });
}
