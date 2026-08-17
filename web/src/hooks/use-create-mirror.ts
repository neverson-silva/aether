import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { RegistryMirror } from "./types";

export function useCreateMirror() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; source: string; dest: string; dest_tls_verify?: boolean; tags_filter?: string; schedule?: string }) =>
      apiPost<RegistryMirror>("/api/v1/mirrors", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["mirrors"] }),
  });
}
