import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { RegistrySettings } from "./types";

export function useToggleRegistry() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (enabled: boolean) => apiPost<RegistrySettings>("/api/v1/registry", { enabled }),
    onSuccess: (data) => {
      qc.setQueryData(["registry"], data);
      qc.invalidateQueries({ queryKey: ["registry", "images"] });
    },
  });
}
