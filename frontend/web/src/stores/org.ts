import { create } from "zustand";
import { getOrgId, setOrgId } from "../api/client";
import type { Me, OrgRoleView } from "../api/types";

interface OrgState {
  orgs: OrgRoleView[];
  currentOrg: OrgRoleView | null;
  role: string;
  isLoading: boolean;
  forcedOrg: string | null;
  setMe: (me: Me | undefined, isLoading: boolean) => void;
  switchOrg: (orgId: string) => void;
  refetch: () => void;
  setRefetch: (fn: () => void) => void;
}

export const useOrgStore = create<OrgState>((set) => ({
  orgs: [],
  currentOrg: null,
  role: "",
  isLoading: true,
  forcedOrg: null,
  setMe: (me, isLoading) =>
    set((state) => {
      const orgs = me?.organizations ?? [];
      const stored = getOrgId();
      const forced = state.forcedOrg;
      const match = stored ? orgs.find((o) => o.id === stored) : undefined;
      const forcedMatch = forced ? orgs.find((o) => o.id === forced) : undefined;
      const currentOrg = match ?? forcedMatch ?? orgs.find((o) => o.id === me?.org?.id) ?? orgs[0] ?? null;
      return {
        orgs,
        currentOrg,
        role: currentOrg?.role ?? me?.org?.role ?? "",
        isLoading,
      };
    }),
  switchOrg: (orgId) => {
    setOrgId(orgId);
    set({ forcedOrg: orgId });
  },
  refetch: () => {},
  setRefetch: (fn) => set({ refetch: fn }),
}));

export function useOrg() {
  const orgs = useOrgStore((s) => s.orgs);
  const currentOrg = useOrgStore((s) => s.currentOrg);
  const role = useOrgStore((s) => s.role);
  const isLoading = useOrgStore((s) => s.isLoading);
  const switchOrg = useOrgStore((s) => s.switchOrg);
  const refetch = useOrgStore((s) => s.refetch);
  return { orgs, currentOrg, role, isLoading, switchOrg, refetch };
}