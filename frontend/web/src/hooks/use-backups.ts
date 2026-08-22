import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Backup } from "../api/types";
import { qk } from "./query-keys";

export function useBackups() {
  return useQuery({ queryKey: qk.backups, queryFn: () => apiGet<Backup[]>("/api/v1/backups") });
}
