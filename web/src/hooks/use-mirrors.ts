import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { RegistryMirror } from "./types";

export function useMirrors() {
  return useQuery({ queryKey: ["mirrors"], queryFn: () => apiGet<RegistryMirror[]>("/api/v1/mirrors") });
}
