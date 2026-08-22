import { useQuery } from "@tanstack/react-query";
import { apiGet } from "../api/client";
import type { OutWebhook } from "./types";

export function useOutWebhooks() {
  return useQuery({ queryKey: ["out-webhooks"], queryFn: () => apiGet<OutWebhook[]>("/api/v1/webhooks") });
}
