import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";
import { qk } from "./query-keys";

export function useDeleteApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/api/v1/apps/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.apps }),
  });
}
