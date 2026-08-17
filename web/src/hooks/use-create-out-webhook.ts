import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { OutWebhook } from "./types";

export function useCreateOutWebhook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { name: string; url: string; secret: string; events: string[] }) =>
      apiPost<OutWebhook>("/api/v1/webhooks", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["out-webhooks"] }),
  });
}
