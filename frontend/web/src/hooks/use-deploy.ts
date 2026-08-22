import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Deployment } from "../api/types";
import { qk } from "./query-keys";

export function useDeploy(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost<Deployment>(`/api/v1/apps/${appID}/deploy`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.deployments(appID) });
      qc.invalidateQueries({ queryKey: qk.apps });
    },
  });
}
