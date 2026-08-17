import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { ComposeStack } from "./types";

export function useComposeStack(id: string) {
  return useQuery({ queryKey: ["compose", id], enabled: !!id, queryFn: () => apiGet<ComposeStack>(`/api/v1/compose/${id}`) });
}
