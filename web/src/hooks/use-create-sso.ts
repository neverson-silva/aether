import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { OIDCProvider } from "./types";

export function useCreateSSO() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; issuer: string; client_id: string; client_secret?: string; scopes?: string }) =>
      apiPost<OIDCProvider>("/api/v1/sso", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sso"] }),
  });
}
