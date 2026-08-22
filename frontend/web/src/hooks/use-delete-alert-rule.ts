import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeleteAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/alerts/rules/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });
}
