import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { StudioObject } from "../api/types";

export function useStudioObjects(dbId: string, schema: string) {
  return useQuery({
    queryKey: ["studio", dbId, "objects", schema],
    queryFn: () => apiGet<StudioObject[]>(`/api/v1/databases/${dbId}/studio/schemas/${encodeURIComponent(schema)}/objects/list`),
    enabled: !!dbId && !!schema,
  });
}
