import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Deployment } from "../api/types";
import { qk } from "./query-keys";

export function useCancelDeployment(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (deploymentID: string) =>
      apiPost<Deployment>(`/api/v1/apps/${appID}/deployments/${deploymentID}/cancel`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.deployments(appID) });
      qc.invalidateQueries({ queryKey: qk.apps });
    },
  });
}
