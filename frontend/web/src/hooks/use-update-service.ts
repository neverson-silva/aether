import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPatch } from "../api/client";

export interface ServiceUpdate {
  name?: string;
  port?: number;
  image_retention?: number;
  resources?: { cpus?: string; mem_mb?: number; storage_mb?: number };
}

export function useUpdateService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ serviceId, update }: { serviceId: string; update: ServiceUpdate }) => apiPatch(`/api/v1/services/${serviceId}`, update),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["service", variables.serviceId] });
      queryClient.invalidateQueries({ queryKey: ["services"] });
    },
  });
}
