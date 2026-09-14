import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export function useInstallTemplate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: { id: string; project_id: string; name?: string; overrides?: Record<string, string> }) =>
      apiPost(`/api/v1/templates/${body.id}/install`, body),
    onSuccess: () => Promise.all([
      queryClient.invalidateQueries({ queryKey: qk.services }),
      queryClient.invalidateQueries({ queryKey: qk.apps }),
      queryClient.invalidateQueries({ queryKey: qk.composes }),
    ]),
  });
}
