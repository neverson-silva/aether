import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Database } from "../api/types";

export function useDatabases() {
  return useQuery({ queryKey: ["databases"], queryFn: () => apiGet<Database[]>("/api/v1/databases") });
}
