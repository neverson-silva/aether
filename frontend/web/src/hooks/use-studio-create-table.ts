import { useMutation } from "@tanstack/react-query";
import { apiPost } from "../api/client";
import type { StudioExecResult } from "../api/types";

export interface CreateTableColumn {
  name: string;
  type: string;
  nullable: boolean;
  primary: boolean;
  default?: string;
}

export function useStudioCreateTable(dbId: string) {
  return useMutation({
    mutationFn: ({ table, schema, columns }: { table: string; schema: string; columns: CreateTableColumn[] }) =>
      apiPost<StudioExecResult>(`/api/v1/databases/${dbId}/studio/tables`, { table, schema, columns }),
  });
}