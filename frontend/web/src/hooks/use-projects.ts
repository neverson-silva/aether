import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Project } from "../api/types";
import { qk } from "./query-keys";

export function useProjects() {
  return useQuery({ queryKey: qk.projects, queryFn: () => apiGet<Project[]>("/api/v1/projects") });
}
