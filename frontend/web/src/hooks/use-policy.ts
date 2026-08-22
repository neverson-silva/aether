import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { AppPolicy } from "./types";

export function usePolicy(appId: string) {
  return useQuery({ queryKey: ["policy", appId], queryFn: () => apiGet<AppPolicy>(`/api/v1/apps/${appId}/policy`) });
}
