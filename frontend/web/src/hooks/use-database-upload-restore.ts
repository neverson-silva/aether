import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiUpload } from "../api/client";
import type { RestoreJob } from "../api/types";

export function useDatabaseUploadRestore(serviceId: string) {
  const prefix = `/api/v1/services/${serviceId}`;
  const queryClient = useQueryClient();

  const createRestore = useMutation({
    mutationFn: (filename: string) =>
      apiPost<RestoreJob>(`${prefix}/restores`, { filename }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", "service", serviceId] }),
  });

  const uploadRestore = useMutation({
    mutationFn: ({
      restoreId,
      file,
      onProgress,
    }: {
      restoreId: string;
      file: File;
      onProgress?: (loaded: number, total: number) => void;
    }) => apiUpload<RestoreJob>(`${prefix}/restores/${restoreId}/upload`, file, onProgress),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", "service", serviceId] }),
  });

  const validateRestore = useMutation({
    mutationFn: (restoreId: string) =>
      apiPost<RestoreJob>(`${prefix}/restores/${restoreId}/validate`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", "service", serviceId] }),
  });

  const startRestore = useMutation({
    mutationFn: (restoreId: string) =>
      apiPost<RestoreJob>(`${prefix}/restores/${restoreId}/start`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", "service", serviceId] }),
  });

  const cancelRestore = useMutation({
    mutationFn: (restoreId: string) =>
      apiDelete<{ ok: boolean }>(`${prefix}/restores/${restoreId}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", "service", serviceId] }),
  });

  return { createRestore, uploadRestore, validateRestore, startRestore, cancelRestore };
}

export function useRestoreJob(serviceId: string, restoreId: string | null) {
  return useQuery({
    queryKey: ["database-restore", "service", serviceId, restoreId],
    queryFn: () => apiGet<RestoreJob>(`/api/v1/services/${serviceId}/restores/${restoreId as string}`),
    enabled: restoreId !== null,
  });
}
