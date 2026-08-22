import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { CertInfo } from "./types";

export function useCertificates() {
  return useQuery({ queryKey: ["certificates"], queryFn: () => apiGet<CertInfo[]>("/api/v1/certificates") });
}
