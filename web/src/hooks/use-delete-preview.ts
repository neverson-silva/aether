import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useDeletePreview(appID: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/previews/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["previews", appID] }),
  });
}
