import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { StudioTableDetail } from "../api/types";

export function useStudioTable(dbId: string, schema: string, table: string, enabled = true) {
  return useQuery({
    queryKey: ["studio", dbId, "table", schema, table],
    queryFn: () => apiGet<StudioTableDetail>(`/api/v1/databases/${dbId}/studio/tables/${encodeURIComponent(schema)}/${encodeURIComponent(table)}`),
    enabled: enabled && !!dbId && !!schema && !!table,
  });
}
