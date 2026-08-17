import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { NotificationChannel } from "../api/types";

export function useCreateChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => apiPost<NotificationChannel>("/api/v1/notification-channels", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["channels"] }),
  });
}
