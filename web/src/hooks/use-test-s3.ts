import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useTestS3() {
  return useMutation({
    mutationFn: (id: string) => apiPost(`/api/v1/s3-destinations/${id}/test`),
  });
}
