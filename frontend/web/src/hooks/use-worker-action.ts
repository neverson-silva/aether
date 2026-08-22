import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useWorkerAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action }: { id: string; action: "start" | "stop" }) =>
      apiPost(`/api/v1/workers/${id}/${action}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["workers"] });
    },
  });
}
