import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { PipelineStage, Pipeline } from "./types";

export function useCreatePipeline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { app_id: string; name: string; trigger: string; stages: PipelineStage[] }) =>
      apiPost<Pipeline>("/api/v1/pipelines", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["pipelines"] }),
  });
}
