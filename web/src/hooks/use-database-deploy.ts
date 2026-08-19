import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Database } from "../api/types";
import { useQueryClient } from "@tanstack/react-query";

export function useDatabaseDeploy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<Database>(`/api/v1/databases/${id}/deploy`),
    onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: ["database", id] }),
  });
}
