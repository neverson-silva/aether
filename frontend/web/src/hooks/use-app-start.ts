import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useAppStart() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (id: string) => apiPost(`/api/v1/apps/${id}/start`), onSuccess: () => qc.invalidateQueries({ queryKey: ["app-states"] }) });
}
