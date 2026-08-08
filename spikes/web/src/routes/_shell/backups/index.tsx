import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useBackups, useCreateBackup, useRestoreBackup } from "../../../api/hooks";
import { AppPageHeader } from "../../../components/ds";
import {
  Button,
  Card,
  ConfirmDialog,
  EmptyState,
  Table,
  fmtBytes,
  fmtDate,
  useToast,
} from "../../../components/ui";

function Backups() {
  const { data: backups, isLoading } = useBackups();
  const createBackup = useCreateBackup();
  const restoreBackup = useRestoreBackup();
  const { toast } = useToast();
  const [confirmId, setConfirmId] = useState<string | null>(null);

  return (
    <div>
      <AppPageHeader
        title="Backups"
        description="Consistent snapshots of platform state (SQLite VACUUM INTO)."
        actions={
          <Button
            leftIcon="backup"
            onClick={() =>
              createBackup.mutate(undefined, {
                onSuccess: () => toast("Backup created"),
                onError: (e) => toast(e.message, "error"),
              })
            }
          >
            Create backup
          </Button>
        }
      />

      {isLoading && <div className="py-md" />}
      {!isLoading && !backups?.length && (
        <EmptyState
          icon="backup"
          title="No backups"
          description="Create scheduled backups to protect platform state."
        />
      )}
      {!!backups?.length && (
        <div className="bg-surface-container-low border border-outline-variant rounded-lg">
          <Table headers={["ID", "Size", "Created", ""]}>
            {backups.map((b) => (
              <tr key={b.id} className="hover:bg-surface-container-high transition-colors">
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface">{b.id}</td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{fmtBytes(b.size)}</td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">{fmtDate(b.created_at)}</td>
                <td className="px-sm py-2 text-right">
                  <button
                    onClick={() => setConfirmId(b.id)}
                    className="flex items-center gap-1 font-label-caps text-label-caps text-tertiary hover:text-tertiary-fixed-dim transition-colors uppercase"
                  >
                    <span className="material-symbols-outlined text-[14px]">restore</span>
                    Restore
                  </button>
                </td>
              </tr>
            ))}
          </Table>
        </div>
      )}

      <ConfirmDialog
        open={!!confirmId}
        onClose={() => setConfirmId(null)}
        onConfirm={() =>
          restoreBackup.mutate(confirmId!, {
            onSuccess: () => toast("Restore completed"),
            onError: (e) => toast(e.message, "error"),
          })
        }
        title="Restore backup"
        description="The current state will be replaced by the backup. Active containers will be stopped and reconciled."
        confirmLabel="Restore"
        danger
      />
    </div>
  );
}

export const Route = createFileRoute('/_shell/backups/')({
  component: Backups,
});
