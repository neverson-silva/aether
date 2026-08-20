import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

export function useStudioSchemas(dbId: string) {
  return useQuery({
    queryKey: ["studio", dbId, "schemas"],
    queryFn: () => apiGet<string[]>(`/api/v1/databases/${dbId}/studio/schemas`),
    enabled: !!dbId,
  });
}
