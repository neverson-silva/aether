import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

export function useTemplateCategories() {
  return useQuery({ queryKey: ["templates-categories"], queryFn: () => apiGet<string[]>("/api/v1/templates?categories=true") });
}
