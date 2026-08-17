import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { NotificationItem } from "./types";

export function useNotifications() {
  return useQuery({ queryKey: ["notifications"], queryFn: () => apiGet<NotificationItem[]>("/api/v1/notifications") });
}
