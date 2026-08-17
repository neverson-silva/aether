import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Me } from "../api/types";
import { qk } from "./query-keys";

export function useMe() {
  return useQuery({ queryKey: qk.me, queryFn: () => apiGet<Me>("/api/v1/me") });
}
