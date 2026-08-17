import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { OrgDetail } from "../api/types";

export function useOrgDetail(orgId: string) {
  return useQuery({
    queryKey: ["org", orgId],
    enabled: !!orgId,
    queryFn: () => apiGet<OrgDetail>(`/api/v1/organizations/${orgId}`),
  });
}
