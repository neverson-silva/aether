import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiDelete } from "../api/client";

export function useClusterRemoveServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { cluster_id: string; server_id: string }) =>
      apiDelete(`/api/v1/clusters/${body.cluster_id}/servers/${body.server_id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clusters"] }),
  });
}
