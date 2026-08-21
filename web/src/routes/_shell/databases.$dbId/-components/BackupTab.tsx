import { useState } from "react";
import type { BackupConfig, BackupJob } from "../../../../api/types";
import {
  useDatabaseBackupCancel,
  useDatabaseBackupConfig,
  useDatabaseBackupNow,
  useDatabaseBackups,
  useDeleteDatabaseBackupConfig,
  useS3Destinations,
} from "../../../../hooks";
import { Button, Card, ConfirmDialog, StatusPill, fmtBytes, useToast } from "../../../../components/ui";
import { BackupConfigDialog, describeScheduleExport } from "./BackupConfigDialog";
import { RestoreDialog } from "./RestoreDialog";

const ACTIVE_STATUSES = ["queued", "preparing", "running", "uploading", "verifying", "cancelling"];

function backupTone(status: string): { status: string; pulse?: boolean } {
  if (status === "completed") return { status: "completed" };
  if (status === "failed") return { status: "failed" };
  if (status === "cancelled") return { status: "cancelled" };
  return { status, pulse: true };
}

export function BackupTab({ dbId, dbName }: { dbId: string; dbName?: string }) {
  const { toast } = useToast();
  const { data: config } = useDatabaseBackupConfig(dbId);
  const { data: backups } = useDatabaseBackups(dbId);
  const { data: destinations } = useS3Destinations();
  const backupNow = useDatabaseBackupNow(dbId);
  const cancel = useDatabaseBackupCancel(dbId);
  const removeConfig = useDeleteDatabaseBackupConfig(dbId);

  const [configOpen, setConfigOpen] = useState(false);
  const [restoreTarget, setRestoreTarget] = useState<BackupJob | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const destName = (id: string) => destinations?.find((d) => d.id === id)?.name ?? id;

  const runNow = () =>
    backupNow.mutate(undefined, {
      onSuccess: () => toast("Backup queued"),
      onError: (e) => toast(e instanceof Error ? e.message : "failed to queue backup", "error"),
    });

  if (!config) {
    return (
      <div className="space-y-lg">
        <Card className="p-xl text-center">
          <span className="material-symbols-outlined text-[48px] text-on-surface-variant/40">cloud_upload</span>
          <h2 className="font-title-md text-title-md text-on-surface mt-md">Automated backups</h2>
          <p className="font-body-md text-body-md text-on-surface-variant max-w-md mx-auto mt-sm">
            Schedule point-in-time backups of this database to an S3 destination. Backups are streamed, checksummed and
            verified after upload.
          </p>
          <div className="mt-lg">
            <Button leftIcon="settings" onClick={() => setConfigOpen(true)} disabled={(destinations ?? []).length === 0}>
              Configure backup
            </Button>
          </div>
          {(destinations ?? []).length === 0 && (
            <p className="font-body-sm text-body-sm text-on-surface-variant/70 mt-md">
              You need at least one S3 destination configured in Settings.
            </p>
          )}
        </Card>
        {configOpen && destinations && (
          <BackupConfigDialog dbId={dbId} existing={null} destinations={destinations} onClose={() => setConfigOpen(false)} />
        )}
      </div>
    );
  }

  const cfg = config as BackupConfig;

  return (
    <div className="space-y-lg">
      <Card className="p-lg">
        <div className="flex flex-wrap items-start justify-between gap-md">
          <div className="min-w-0">
            <div className="flex items-center gap-sm mb-sm">
              <h2 className="font-title-md text-title-md text-on-surface">Backup schedule</h2>
              <StatusPill status={cfg.enabled ? "active" : "disabled"} />
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-md mt-md">
              <div>
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Schedule</span>
                <span className="font-body-md text-body-md text-on-surface">{describeScheduleExport(cfg.schedule)}</span>
              </div>
              <div>
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Destination</span>
                <span className="font-body-md text-body-md text-on-surface truncate">{destName(cfg.destination_id)}</span>
              </div>
              <div>
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Retention</span>
                <span className="font-body-md text-body-md text-on-surface">{cfg.retention.type === "latest" ? "Latest only" : "Keep all"}</span>
              </div>
              <div>
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Next run</span>
                <span className="font-body-md text-body-md text-on-surface">
                  {cfg.next_run_at ? new Date(cfg.next_run_at).toLocaleString() : "—"}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-sm shrink-0">
            <Button variant="outline" size="sm" leftIcon="tune" onClick={() => setConfigOpen(true)}>
              Edit
            </Button>
            <Button variant="ghost" size="sm" leftIcon="delete" onClick={() => setDeleteOpen(true)}>
              Remove
            </Button>
            <Button leftIcon="backup" onClick={runNow} loading={backupNow.isPending}>
              Backup now
            </Button>
          </div>
        </div>
      </Card>

      <Card className="overflow-hidden">
        <div className="flex items-center justify-between px-lg py-md border-b border-outline-variant/60">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Backup history</h2>
        </div>
        {(backups ?? []).length === 0 ? (
          <p className="px-lg py-xl font-body-md text-body-md text-on-surface-variant text-center">
            No backups yet. Run your first backup with “Backup now”.
          </p>
        ) : (
          <div className="divide-y divide-outline-variant/40">
            {(backups ?? []).map((b) => {
              const active = ACTIVE_STATUSES.includes(b.status);
              return (
                <div key={b.id} className="flex flex-wrap items-center gap-md px-lg py-md hover:bg-surface-container/50 transition-colors">
                  <StatusPill status={backupTone(b.status).status} pulse={backupTone(b.status).pulse} />
                  <div className="min-w-0 flex-1">
                    <p className="font-code-md text-code-md text-on-surface truncate">
                      {new Date(b.started_at ?? "").toLocaleString()} · {b.trigger}
                    </p>
                    <p className="font-code-md text-code-md text-on-surface-variant/60 truncate">
                      {b.format.toUpperCase()}
                      {b.size_bytes > 0 && ` · ${fmtBytes(b.size_bytes)}`}
                      {b.checksum && ` · sha256 ${b.checksum.slice(0, 12)}…`}
                      {b.error_message && ` · ${b.error_code}: ${b.error_message}`}
                    </p>
                  </div>
                  <div className="flex items-center gap-sm shrink-0">
                    {active && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          cancel.mutate(b.id, {
                            onSuccess: () => toast("Cancellation requested"),
                            onError: (e) => toast(e instanceof Error ? e.message : "cancel failed", "error"),
                          })
                        }
                      >
                        Cancel
                      </Button>
                    )}
                    {b.status === "completed" && (
                      <Button variant="outline" size="sm" leftIcon="restore" onClick={() => setRestoreTarget(b)}>
                        Restore
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </Card>

      {configOpen && destinations && (
        <BackupConfigDialog dbId={dbId} existing={cfg} destinations={destinations} onClose={() => setConfigOpen(false)} />
      )}
      {restoreTarget && <RestoreDialog dbId={dbId} dbName={dbName} backup={restoreTarget} onClose={() => setRestoreTarget(null)} />}
      {deleteOpen && (
        <ConfirmDialog
          open
          title="Remove backup configuration"
          description="Scheduled backups will stop for this database. Existing backups in storage are kept."
          confirmLabel="Remove"
          danger
          onConfirm={() =>
            removeConfig.mutate(undefined, {
              onSuccess: () => toast("Backup configuration removed"),
              onError: (e) => toast(e instanceof Error ? e.message : "remove failed", "error"),
            })
          }
          onClose={() => setDeleteOpen(false)}
        />
      )}
    </div>
  );
}