import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useVolumeBackup() {
  return useMutation({
    mutationFn: ({ serviceID, name, destination_id }: { serviceID: string; name: string; destination_id: string }) =>
      apiPost(`/api/v1/services/${serviceID}/volumes/${name}/backup`, { destination_id }),
  });
}
