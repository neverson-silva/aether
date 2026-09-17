import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { OrgMember } from "../api/types";

export function useOrgMembers(orgId: string) {
  return useQuery({
    queryKey: ["org-members", orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const response = await apiGet<OrgMember[] | { members?: OrgMember[] }>(
        `/api/v1/organizations/${orgId}/members`,
      );
      return Array.isArray(response) ? response : (response.members ?? []);
    },
  });
}
