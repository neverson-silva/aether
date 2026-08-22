import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useResolveAlert() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/alerts/events/${id}/resolve`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-events"] }),
  });
}
