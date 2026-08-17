import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { S3Destination } from "../api/types";

export function useCreateS3() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => apiPost<S3Destination>("/api/v1/s3-destinations", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["s3"] }),
  });
}
