import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { NetQStat } from "./types";

export function useNetQ() {
  return useQuery({ queryKey: ["netq"], queryFn: () => apiGet<NetQStat[]>("/api/v1/network/quality"), refetchInterval: 15000 });
}
