import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { AlertRule } from "./types";

export function useCreateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<AlertRule>) => apiPost("/api/v1/alerts/rules", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });
}
