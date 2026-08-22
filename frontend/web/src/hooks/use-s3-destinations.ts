import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { S3Destination } from "../api/types";

export function useS3Destinations() {
  return useQuery({ queryKey: ["s3"], queryFn: () => apiGet<S3Destination[]>("/api/v1/s3-destinations") });
}
