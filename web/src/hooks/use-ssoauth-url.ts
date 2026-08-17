import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useSSOAuthURL() {
  return useMutation({
    mutationFn: (id: string) => apiPost<{ url: string }>(`/api/v1/sso/${id}/auth-url`),
  });
}
