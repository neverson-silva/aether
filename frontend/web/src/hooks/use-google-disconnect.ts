import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useGoogleDisconnect() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/s3-destinations/${id}/google/disconnect`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["s3"] }),
  });
}
