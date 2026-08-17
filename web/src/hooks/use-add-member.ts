import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import { qk } from "./query-keys";

export function useAddMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { email: string; password: string; name: string; role: string }) =>
      apiPost("/api/v1/members", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.members }),
  });
}
