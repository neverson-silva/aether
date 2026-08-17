import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { PipelineRun } from "./types";

export function useRunPipeline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<PipelineRun>(`/api/v1/pipelines/${id}/run`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["pipeline-runs"] }),
  });
}
