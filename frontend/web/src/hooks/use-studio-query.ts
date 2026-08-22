import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { StudioQueryResult } from "../api/types";

export function useStudioQuery(dbId: string) {
  return useMutation({
    mutationFn: (sql: string) => apiPost<StudioQueryResult>(`/api/v1/databases/${dbId}/studio/query`, { sql }),
  });
}
