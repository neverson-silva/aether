import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPatch } from "../api/client";

export function useSetAlertRuleEnabled() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { id: string; enabled: boolean }) => apiPatch(`/api/v1/alerts/rules/${body.id}`, { enabled: body.enabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });
}
