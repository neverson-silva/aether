import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; scopes: string[] }) =>
      apiPost<{ key: string }>("/api/v1/api-keys", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.keys }),
  });
}
