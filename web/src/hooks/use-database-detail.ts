import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Database } from "../api/types";

export function useDatabaseDetail(dbId: string) {
  return useQuery({
    queryKey: ["database", dbId],
    queryFn: () => apiGet<{ database: Database; dsn: string }>(`/api/v1/databases/${dbId}`),

  });
}
