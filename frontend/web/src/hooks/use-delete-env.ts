import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";
import { qk } from "./query-keys";

export function useDeleteEnv(appID: string, serviceScoped = false) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => apiDelete(`/api/v1/${serviceScoped ? "services" : "apps"}/${appID}/${serviceScoped ? "environment" : "env"}/${name}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: serviceScoped ? qk.serviceEnvironment(appID) : qk.app(appID) }),
  });
}
