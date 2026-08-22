import { createFileRoute } from "@tanstack/react-router";
import { useBackups, useCreateBackup, useRestoreBackup } from "../../../hooks";
import { Archive, ArrowUUpLeft } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { AlertDialog, Button, Card, EmptyState, Skeleton, useToast } from "@aether/design-system";

function fmtBytes(size: number) { if (size < 1024) return `${size} B`; if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KiB`; return `${(size / (1024 * 1024)).toFixed(1)} MiB`; }
function fmtDate(iso: string) { return new Intl.DateTimeFormat("en", { dateStyle: "medium", timeStyle: "short" }).format(new Date(iso)); }

function Backups() {
  const { data: backups, isLoading } = useBackups();
  const createBackup = useCreateBackup();
  const restoreBackup = useRestoreBackup();
  const { add } = useToast();

  return (
    <div>
      <header className="flex flex-wrap items-end justify-between gap-4"><div><p className="text-label-caps text-primary">Operations</p><h1 className="text-headline-sm font-semibold text-foreground">Backups</h1><p className="mt-1 text-body-md text-muted-foreground">Consistent snapshots of platform state.</p></div><Button icon={Archive as unknown as DesignIcon} loading={createBackup.isPending} onClick={() => createBackup.mutate(undefined, { onSuccess: () => add({ title: "Backup created", tone: "success" }), onError: (error) => add({ title: "Could not create backup", description: error.message, tone: "error" }) })}>Create backup</Button></header>

      {isLoading && <div className="space-y-sm py-md" aria-label="Loading backups"><Skeleton variant="table" /><Skeleton variant="table" /><Skeleton variant="table" /></div>}
      {!isLoading && !backups?.length && (
        <EmptyState
          icon={Archive as unknown as DesignIcon}
          title="No backups"
          description="Create scheduled backups to protect platform state."
        />
      )}
      {!!backups?.length && (
        <Card padding="none"><div className="overflow-x-auto"><table className="w-full text-left"><thead><tr className="border-b border-border text-label-caps text-muted-foreground">{["ID", "Size", "Created", ""].map((header) => <th key={header} className="px-3 py-3">{header}</th>)}</tr></thead><tbody>
            {backups.map((b) => (
              <tr key={b.id} className="hover:bg-surface-container-high transition-colors">
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface">{b.id}</td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{fmtBytes(b.size)}</td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">{fmtDate(b.created_at)}</td>
                <td className="px-sm py-2 text-right">
                    <AlertDialog trigger={<Button variant="quiet" size="sm" icon={ArrowUUpLeft as unknown as DesignIcon}>Restore</Button>} title="Restore backup" description="The current state will be replaced by this backup. Active containers will be stopped and reconciled." confirmLabel="Restore" onConfirm={() => restoreBackup.mutate(b.id, { onSuccess: () => add({ title: "Restore completed", tone: "success" }), onError: (error) => add({ title: "Restore failed", description: error.message, tone: "error" }) })} />
                </td>
              </tr>
            ))}
          </tbody></table></div></Card>
      )}

    </div>
  );
}

export const Route = createFileRoute('/_shell/backups/')({
  component: Backups,
});
