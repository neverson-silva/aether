import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getOrgId, setOrgId } from "../api/client";
import { useMe } from "../hooks";
import type { OrgRoleView } from "../api/types";

interface OrgContextValue {
  orgs: OrgRoleView[];
  currentOrg: OrgRoleView | null;
  role: string;
  isLoading: boolean;
  switchOrg: (orgId: string) => void;
  refetch: () => void;
}

const OrgContext = createContext<OrgContextValue>({
  orgs: [],
  currentOrg: null,
  role: "",
  isLoading: true,
  switchOrg: () => {},
  refetch: () => {},
});

export function useOrg() {
  return useContext(OrgContext);
}

export function OrgProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const { data, isLoading, refetch } = useMe();
  const [forced, setForced] = useState<string | null>(null);

  const orgs = data?.organizations ?? [];

  const currentOrg = useMemo<OrgRoleView | null>(() => {
    const stored = getOrgId();
    const match = stored
      ? orgs.find((o) => o.id === stored)
      : null;
    if (match) return match;
    if (forced) {
      const f = orgs.find((o) => o.id === forced);
      if (f) return f;
    }
    return orgs.find((o) => o.id === data?.org?.id) ?? orgs[0] ?? null;
  }, [orgs, forced, data]);

  useEffect(() => {
    if (currentOrg && !getOrgId()) {
      setOrgId(currentOrg.id);
    }
  }, [currentOrg]);

  const switchOrg = (orgId: string) => {
    setOrgId(orgId);
    setForced(orgId);
    queryClient.invalidateQueries();
  };

  const value: OrgContextValue = {
    orgs,
    currentOrg,
    role: currentOrg?.role ?? data?.org?.role ?? "",
    isLoading,
    switchOrg,
    refetch,
  };

  return <OrgContext.Provider value={value}>{children}</OrgContext.Provider>;
}
