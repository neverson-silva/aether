import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Deployment } from "../api/types";
import { qk } from "./query-keys";

export function useServiceCancelDeployment(serviceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (deploymentId: string) => apiPost<Deployment>(`/api/v1/services/${serviceId}/deployments/${deploymentId}/cancel`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk.serviceDeployments(serviceId) }),
  });
}
