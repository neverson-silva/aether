import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { HostInfo } from "../api/types";

export function useHostInfo() {
  return useQuery({
    queryKey: ["host-info"],
    queryFn: () => apiGet<HostInfo>("/api/v1/host/info"),
  });
}