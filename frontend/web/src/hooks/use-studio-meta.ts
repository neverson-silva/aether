import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { StudioMeta } from "../api/types";

export function useStudioMeta(dbId: string) {
  return useQuery({
    queryKey: ["studio", dbId, "meta"],
    queryFn: () => apiGet<StudioMeta>(`/api/v1/databases/${dbId}/studio/meta`),
    enabled: !!dbId,
  });
}
