import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPut } from "../api/client";
import { qk } from "./query-keys";

export function useSetEnv(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; value: string; secret: boolean }) =>
      apiPut(`/api/v1/apps/${appID}/env`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.app(appID) }),
  });
}
