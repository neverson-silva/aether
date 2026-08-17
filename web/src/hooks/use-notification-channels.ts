import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { NotificationChannel } from "../api/types";

export function useNotificationChannels() {
  return useQuery({ queryKey: ["channels"], queryFn: () => apiGet<NotificationChannel[]>("/api/v1/notification-channels") });
}
