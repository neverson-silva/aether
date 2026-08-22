import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";

export function useVolumeBackup() {
  return useMutation({
    mutationFn: ({ appID, name, destination_id }: { appID: string; name: string; destination_id: string }) =>
      apiPost(`/api/v1/apps/${appID}/volumes/${name}/backup`, { destination_id }),
  });
}
