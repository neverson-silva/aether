import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { RegistrySettings } from "./types";

export function useRegistrySettings() {
  return useQuery({ queryKey: ["registry"], queryFn: () => apiGet<RegistrySettings>("/api/v1/registry") });
}
