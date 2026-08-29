import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { Database } from "../api/types";

export function useCreateDatabase() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { project_id: string; environment_id?: string; name: string; engine: string; version?: string; user?: string; password?: string; mem_mb?: number; storage_mb?: number }) =>
      apiPost<Database>("/api/v1/databases", body),
    onSuccess: () => Promise.all([
        qc.invalidateQueries({ queryKey: ["databases"] }),
        qc.invalidateQueries({ queryKey: ["services"] }),
      ]),
  });
}
