import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPut } from "../api/client";
import { qk } from "./query-keys";

export function useUpdateMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ userID, role }: { userID: string; role: string }) =>
      apiPut(`/api/v1/members/${userID}`, { role }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.members }),
  });
}
