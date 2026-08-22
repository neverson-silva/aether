import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { OrgMember } from "../api/types";

export function useOrgMembers(orgId: string) {
  return useQuery({
    queryKey: ["org-members", orgId],
    enabled: !!orgId,
    queryFn: () => apiGet<OrgMember[]>(`/api/v1/organizations/${orgId}/members`),
  });
}
