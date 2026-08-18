import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPatch } from "../api/client";
import type { S3Destination } from "../api/types";

export function useUpdateS3() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: unknown }) => apiPatch<S3Destination>(`/api/v1/s3-destinations/${id}`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["s3"] }),
  });
}
