import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useClusterAddServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { cluster_id: string; server_id: string }) =>
      apiPost(`/api/v1/clusters/${body.cluster_id}/servers`, { server_id: body.server_id }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clusters"] }),
  });
}
