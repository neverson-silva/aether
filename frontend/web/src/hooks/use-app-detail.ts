import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { AppDetail } from "../api/types";
import { qk } from "./query-keys";

export function useAppDetail(id: string) {
  return useQuery({
    queryKey: qk.app(id),
    queryFn: () => apiGet<AppDetail>(`/api/v1/apps/${id}`),
    enabled: !!id,
  });
}
