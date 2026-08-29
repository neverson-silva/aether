import { useMutation } from "@tanstack/react-query";
import { apiPut } from "../api/client";

export function useSetWebhook(appID: string, canonical = false) {
  return useMutation({
    mutationFn: (secret: string) => apiPut(canonical ? `/api/v1/services/${appID}/webhook` : `/api/v1/apps/${appID}/webhook`, { secret }),
  });
}
