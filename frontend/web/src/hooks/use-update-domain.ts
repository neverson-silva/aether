import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPatch } from "../api/client";
import type { Domain } from "../api/types";
import { qk } from "./query-keys";

export interface DomainInput {
  host: string;
  https: boolean;
  container_port?: number;
  path?: string;
  internal_path?: string;
  strip_path?: boolean;
  compose_service_name?: string;
}

export function useUpdateDomain(kind: string, id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ domainID, body }: { domainID: string; body: DomainInput }) =>
      apiPatch<Domain>(`/api/v1/${kind}/${id}/domains/${domainID}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.domains(kind, id) }),
  });
}
