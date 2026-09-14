import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { ServiceContainer } from "./use-service-containers";
import type { ServiceSummary, Stats } from "../api/types";
import { qk } from "./query-keys";

export type ServiceAction = "deploy" | "start" | "stop" | "restart" | "delete";

type ServiceActionResponse = {
  result?: unknown;
};

export function useServiceAction(action: ServiceAction) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (serviceId: string) => apiPost<ServiceActionResponse>(`/api/v1/services/${serviceId}/${action}`),
    onSuccess: (response, serviceId) => {
      const result = typeof response?.result === "string" ? response.result : "";
      const state = action === "start"
        ? result === "deploying" ? "deploying" : "running"
        : action === "stop" ? "stopped" : "";
      if (state) {
        queryClient.setQueryData<ServiceSummary>(qk.service(serviceId), (current) => current ? { ...current, status: state as ServiceSummary["status"] } : current);
        queryClient.setQueryData<Stats>(qk.serviceStats(serviceId), (current) => current ? { ...current, state } : current);
        if (state !== "deploying") {
          queryClient.setQueryData<ServiceContainer[]>(qk.serviceContainers(serviceId), (current) => current?.map((container) => ({ ...container, status: state === "running" ? "running" : "exited" })));
        }
      }
      void queryClient.invalidateQueries({ queryKey: qk.services });
      void queryClient.invalidateQueries({ queryKey: qk.service(serviceId) });
      void queryClient.invalidateQueries({ queryKey: qk.serviceStats(serviceId) });
      void queryClient.invalidateQueries({ queryKey: qk.serviceContainers(serviceId) });
      void queryClient.invalidateQueries({ queryKey: qk.serviceDeployments(serviceId) });
    },
  });
}
