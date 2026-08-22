import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { ComposeStack } from "./types";
import { qk } from "./query-keys";

export function useComposeStacks() {
  return useQuery({ queryKey: qk.composes, queryFn: () => apiGet<ComposeStack[]>("/api/v1/compose") });
}
