import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useGoogleConnect() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await apiPost<{ auth_url: string }>(`/api/v1/s3-destinations/${id}/google/connect`);
      window.location.assign(res.auth_url);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: ["s3"] }),
  });
}
