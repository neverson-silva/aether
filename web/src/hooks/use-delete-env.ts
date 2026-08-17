import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";
import { qk } from "./query-keys";

export function useDeleteEnv(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => apiDelete(`/api/v1/apps/${appID}/env/${name}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.app(appID) }),
  });
}
