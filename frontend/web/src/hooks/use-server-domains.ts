import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

export interface ServerDomainSettings {
  web_domain: string;
  api_domain: string;
  https: boolean;
}

export function useServerDomains() {
  return useQuery({
    queryKey: ["server-domains"],
    queryFn: () => apiGet<ServerDomainSettings>("/api/v1/admin/server-domains"),
  });
}
