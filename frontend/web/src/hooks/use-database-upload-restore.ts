import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiUpload } from "../api/client";
import type { RestoreJob } from "../api/types";

export function useDatabaseUploadRestore(dbId: string) {
  const queryClient = useQueryClient();

  const createRestore = useMutation({
    mutationFn: (filename: string) =>
      apiPost<RestoreJob>(`/api/v1/databases/${dbId}/restores`, { filename }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", dbId] }),
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
    }) => apiUpload<RestoreJob>(`/api/v1/databases/${dbId}/restores/${restoreId}/upload`, file, onProgress),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", dbId] }),
  });

  const validateRestore = useMutation({
    mutationFn: (restoreId: string) =>
      apiPost<RestoreJob>(`/api/v1/databases/${dbId}/restores/${restoreId}/validate`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", dbId] }),
  });

  const startRestore = useMutation({
    mutationFn: (restoreId: string) =>
      apiPost<RestoreJob>(`/api/v1/databases/${dbId}/restores/${restoreId}/start`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", dbId] }),
  });

  const cancelRestore = useMutation({
    mutationFn: (restoreId: string) =>
      apiDelete<{ ok: boolean }>(`/api/v1/databases/${dbId}/restores/${restoreId}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", dbId] }),
  });

  return { createRestore, uploadRestore, validateRestore, startRestore, cancelRestore };
}

export function useRestoreJob(dbId: string, restoreId: string | null) {
  return useQuery({
    queryKey: ["database-restore", dbId, restoreId],
    queryFn: () => apiGet<RestoreJob>(`/api/v1/databases/${dbId}/restores/${restoreId as string}`),
    enabled: restoreId !== null,
    refetchInterval: 1500,
  });
}
