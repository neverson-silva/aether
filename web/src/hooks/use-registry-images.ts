import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { RegistryImage } from "./types";

export function useRegistryImages() {
  return useQuery({
    queryKey: ["registry", "images"],
    queryFn: () => apiGet<RegistryImage[]>("/api/v1/registry/images"),
    enabled: false,
  });
}
