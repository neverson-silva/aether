import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { PipelineRun } from "./types";

export function usePipelineRuns(pipelineId: string) {
  return useQuery({
    queryKey: ["pipeline-runs", pipelineId],
    queryFn: () => apiGet<PipelineRun[]>(`/api/v1/pipelines/${pipelineId}/runs`),
  });
}
