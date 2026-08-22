import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { TemplateItem } from "./types";

export function useTrendingTemplates() {
  return useQuery({ queryKey: ["templates-trending"], queryFn: () => apiGet<TemplateItem[]>("/api/v1/templates?trending=true") });
}
