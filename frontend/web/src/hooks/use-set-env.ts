import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPut } from "../api/client";
import { qk } from "./query-keys";

export function useSetEnv(appID: string, serviceScoped = false) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; value: string; secret: boolean }) =>
      apiPut(`/api/v1/${serviceScoped ? "services" : "apps"}/${appID}/${serviceScoped ? "environment" : "env"}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: serviceScoped ? qk.serviceEnvironment(appID) : qk.app(appID) }),
  });
}
