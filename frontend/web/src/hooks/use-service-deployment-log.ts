import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { DeploymentLog } from "./types";

export function useServiceDeploymentLog(serviceId: string, deploymentId: string | null) {
  return useQuery({
    queryKey: ["service-deploy-log", serviceId, deploymentId],
    enabled: !!serviceId && !!deploymentId,
    queryFn: () => apiGet<DeploymentLog>(`/api/v1/services/${serviceId}/deployments/${deploymentId}/log`),
  });
}
