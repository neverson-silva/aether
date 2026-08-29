import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Database } from "../api/types";

export function useDatabaseDetail(dbId: string, enabled = true) {
  return useQuery({
    queryKey: ["database", dbId],
    queryFn: () => apiGet<{ database: Database; dsn: string; public_host: string }>(`/api/v1/databases/${dbId}`),
    enabled: enabled && !!dbId,
  });
}
