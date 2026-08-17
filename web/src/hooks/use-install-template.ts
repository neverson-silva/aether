import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useInstallTemplate() {
  return useMutation({
    mutationFn: (body: { id: string; project_id: string; name?: string }) =>
      apiPost(`/api/v1/templates/${body.id}/install`, body),
  });
}
