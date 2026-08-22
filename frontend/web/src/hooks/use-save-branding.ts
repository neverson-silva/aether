import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPut } from "../api/client";
import type { Branding } from "./types";

export function useSaveBranding() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: Partial<Branding>) => apiPut<Branding>("/api/v1/branding", b),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["branding"] }),
  });
}
