import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { AuditLog } from "../api/types";

export function useOrgAudit(orgId: string) {
  return useQuery({
    queryKey: ["org-audit", orgId],
    enabled: !!orgId,
    queryFn: () => apiGet<AuditLog[]>(`/api/v1/organizations/${orgId}/audit`),
  });
}
