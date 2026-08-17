import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Cluster } from "./types";

export function useCreateCluster() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; labels?: string[] }) => apiPost<Cluster>("/api/v1/clusters", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["clusters"] }),
  });
}
