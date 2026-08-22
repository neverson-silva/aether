import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Pipeline } from "./types";

export function usePipelines() {
  return useQuery({ queryKey: ["pipelines"], queryFn: () => apiGet<Pipeline[]>("/api/v1/pipelines") });
}
