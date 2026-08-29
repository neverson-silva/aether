import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export type ServiceAction = "deploy" | "start" | "stop" | "restart" | "delete";

export function useServiceAction(action: ServiceAction) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (serviceId: string) => apiPost(`/api/v1/services/${serviceId}/${action}`),
    onSuccess: (_, serviceId) => {
      void queryClient.invalidateQueries({ queryKey: qk.services });
      void queryClient.invalidateQueries({ queryKey: qk.service(serviceId) });
      void queryClient.invalidateQueries({ queryKey: qk.serviceStats(serviceId) });
      void queryClient.invalidateQueries({ queryKey: qk.serviceContainers(serviceId) });
      void queryClient.invalidateQueries({ queryKey: qk.serviceDeployments(serviceId) });
    },
  });
}
