import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useBackupDatabase() {
  return useMutation({ mutationFn: (id: string) => apiPost(`/api/v1/databases/${id}/backup`) });
}
