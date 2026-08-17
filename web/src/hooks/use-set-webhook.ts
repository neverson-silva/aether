import { useMutation } from "@tanstack/react-query";
import { apiPut } from "../api/client";

export function useSetWebhook(appID: string) {
  return useMutation({
    mutationFn: (secret: string) => apiPut(`/api/v1/apps/${appID}/webhook`, { secret }),
  });
}
