import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { StudioExecResult } from "../api/types";
import type { CreateTableColumn } from "./use-studio-create-table";

export function useStudioRenameTable(dbId: string) {
  return useMutation({
    mutationFn: ({ schema, table, name }: { schema: string; table: string; name: string }) =>
      apiPost<StudioExecResult>(`/api/v1/databases/${dbId}/studio/tables/rename`, { schema, table, name }),
  });
}

export function useStudioDropTable(dbId: string) {
  return useMutation({
    mutationFn: ({ schema, table }: { schema: string; table: string }) =>
      apiPost<StudioExecResult>(`/api/v1/databases/${dbId}/studio/tables/drop`, { schema, table }),
  });
}

export function useStudioAlterTable(dbId: string) {
  return useMutation({
    mutationFn: ({ schema, table, columns }: { schema: string; table: string; columns: CreateTableColumn[] }) =>
      apiPost<StudioExecResult>(`/api/v1/databases/${dbId}/studio/tables/alter`, { schema, table, columns }),
  });
}