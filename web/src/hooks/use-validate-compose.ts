import { useQuery } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { ComposeValidation } from "./types";

export function useValidateCompose(content: string) {
  return useQuery({
    queryKey: ["compose-validate", content],
    enabled: content.trim().length > 5,
    queryFn: () => apiPost<ComposeValidation>("/api/v1/compose/validate", { content }),
    staleTime: 500,
  });
}
