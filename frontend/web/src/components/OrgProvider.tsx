import { useEffect, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useMe } from "../hooks";
import { useOrgStore } from "../stores/org";

export function useOrg() {
  const orgs = useOrgStore((s) => s.orgs);
  const currentOrg = useOrgStore((s) => s.currentOrg);
  const role = useOrgStore((s) => s.role);
  const isLoading = useOrgStore((s) => s.isLoading);
  const switchOrg = useOrgStore((s) => s.switchOrg);
  const refetch = useOrgStore((s) => s.refetch);
  return { orgs, currentOrg, role, isLoading, switchOrg, refetch };
}

export function OrgProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const { data, isLoading, refetch } = useMe();

  useEffect(() => {
    useOrgStore.getState().setRefetch(() => refetch());
  }, [refetch]);

  useEffect(() => {
    useOrgStore.getState().setMe(data, isLoading);
  }, [data, isLoading]);

  useEffect(() => {
    const unsub = useOrgStore.subscribe((state) => {
      if (state.forcedOrg) {
        queryClient.invalidateQueries();
      }
    });
    return unsub;
  }, [queryClient]);

  return <>{children}</>;
}