import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { StudioExecResult } from "../api/types";

export function useStudioExec(dbId: string) {
  return useMutation({
    mutationFn: (sql: string) => apiPost<StudioExecResult>(`/api/v1/databases/${dbId}/studio/exec`, { sql }),
  });
}
