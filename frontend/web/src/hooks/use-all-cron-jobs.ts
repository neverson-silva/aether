import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { AllCronJob } from "./types";

export function useAllCronJobs() {
  return useQuery({ queryKey: ["cron-jobs-all"], queryFn: () => apiGet<AllCronJob[]>("/api/v1/cron-jobs") });
}
