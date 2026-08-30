import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { App } from "../api/types";
import { qk } from "./query-keys";

export function useCreateApp() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { projectID: string; payload: Partial<App> }) =>
      apiPost<App>(`/api/v1/projects/${body.projectID}/apps`, body.payload),
    onSuccess: () => Promise.all([
      qc.invalidateQueries({ queryKey: qk.apps }),
      qc.invalidateQueries({ queryKey: qk.services }),
    ]),
  });
}
