import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { GitOpsConfig } from "./types";

export function useGitOps() {
  return useQuery({ queryKey: ["gitops"], queryFn: () => apiGet<GitOpsConfig[]>("/api/v1/gitops") });
}
