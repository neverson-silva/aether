import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { Branding } from "./types";

export function useBranding() {
  return useQuery({ queryKey: ["branding"], queryFn: () => apiGet<Branding>("/api/v1/branding") });
}
