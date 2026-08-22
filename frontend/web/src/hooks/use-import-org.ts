import { useMutation } from "@tanstack/react-query";
import { getServer, ApiError } from "../api/client";

export function useImportOrg() {
  return useMutation({
    mutationFn: async (yaml: string) => {
      const server = getServer();
      const res = await fetch(server + "/api/v1/org/import", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/yaml" },
        body: yaml,
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new ApiError(res.status, body.error || "import falhou");
      }
    },
  });
}
