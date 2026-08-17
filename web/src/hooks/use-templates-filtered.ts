import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { TemplateItem } from "./types";

export function useTemplatesFiltered(params: { category?: string; q?: string; featured?: boolean }) {
  const qs = new URLSearchParams();
  if (params.category) qs.set("category", params.category);
  if (params.q) qs.set("q", params.q);
  if (params.featured) qs.set("featured", "true");
  return useQuery({
    queryKey: ["templates", params],
    queryFn: () => apiGet<TemplateItem[]>(`/api/v1/templates${qs.toString() ? "?" + qs.toString() : ""}`),
  });
}
