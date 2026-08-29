import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { ComposeStack } from "./types";

export function useComposeStack(id: string, enabled = true) {
  return useQuery({ queryKey: ["compose", id], enabled: enabled && !!id, queryFn: () => apiGet<ComposeStack>(`/api/v1/compose/${id}`) });
}
