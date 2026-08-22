import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { TemplateItem } from "./types";

export function useTemplates() {
  return useQuery({ queryKey: ["templates"], queryFn: () => apiGet<TemplateItem[]>(`/api/v1/templates`) });
}
