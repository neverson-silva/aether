import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";

export function useUnreadCount() {
  return useQuery({ queryKey: ["notifications-unread"], queryFn: () => apiGet<{ count: number }>("/api/v1/notifications/unread-count") });
}
