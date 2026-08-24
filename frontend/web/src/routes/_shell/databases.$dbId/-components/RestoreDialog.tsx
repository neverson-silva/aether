import { useState } from "react";
import type { BackupJob, RestoreJob } from "../../../../api/types";
import { useDatabaseBackupPreflight, useDatabaseBackupRestore } from "../../../../hooks";
import { Button, Dialog, Field, Input, useToast } from "@aether/design-system";
import { CheckCircle, CircleNotch, XCircle } from "@phosphor-icons/react";

export function RestoreDialog({
  dbId,
  backup,
  dbName,
  onClose,
}: {
  dbId: string;
  backup: BackupJob;
  dbName?: string;
  onClose: () => void;
}) {
  const [confirm, setConfirm] = useState("");
  const [restoreResult, setRestoreResult] = useState<RestoreJob | null>(null);
  const [restoreError, setRestoreError] = useState<string | null>(null);
  const restore = useDatabaseBackupRestore(dbId);
  const { add } = useToast();
  const { data: preflight } = useDatabaseBackupPreflight(dbId, backup.id, true);
  const [restoring, setRestoring] = useState(false);

  const ready = preflight?.ready ?? false;
  const nameMatches = !!dbName && confirm.trim() === dbName;

  const run = async () => {
    setRestoring(true);
    setRestoreError(null);
		try {
			const result = await restore.mutateAsync(backup.id);
			setRestoreResult(result);
			add({ title: result.status === "completed" ? "Restore completed" : "Restore queued", tone: "success" });
			onClose();
    } catch (error) {
      const message = error instanceof Error ? error.message : "Restore failed";
      setRestoreError(message);
      add({ title: "Restore failed", description: message, tone: "error" });
    } finally {
      setRestoring(false);
    }
  };

  return (
    <Dialog open trigger={<span />} onOpenChange={(open) => { if (!open && !restoring) onClose(); }} title="Restore backup">
      <div className="space-y-lg">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-md px-md py-sm rounded border border-outline-variant/60">
          <div>
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Backup</span>
            <span className="font-code-md text-code-md text-on-surface">{new Date(backup.completed_at ?? backup.started_at ?? "").toLocaleString()}</span>
          </div>
          <div>
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Database</span>
            <span className="font-code-md text-code-md text-on-surface">{dbName ?? dbId}</span>
          </div>
          <div>
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Engine</span>
            <span className="font-code-md text-code-md text-on-surface">{backup.engine}</span>
          </div>
        </div>

        {preflight && (
          <div className="space-y-sm">
            <h3 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Preflight</h3>
            {preflight.checks.map((c) => (
              <div key={c.name} className="flex items-center gap-sm px-md py-sm rounded border border-outline-variant/60">
                {c.ok ? <CheckCircle size={18} className="text-success" /> : <XCircle size={18} className="text-error" />}
                <span className="font-body-md text-body-md text-on-surface">{c.name}</span>
                <span className="ml-auto font-code-md text-code-md text-on-surface-variant/70 truncate">{c.message}</span>
              </div>
            ))}
          </div>
        )}

        <div className="p-md rounded-lg border border-warning/40 bg-warning/10">
          <p className="font-body-md text-body-md text-on-surface font-medium">Restoring will replace database data.</p>
          <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">
            This operation may overwrite existing data and cannot be automatically undone. Type the database name to continue.
          </p>
        </div>

        <Field label={`Type "${dbName ?? "database"}" to confirm`}>
          <Input value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder={dbName} disabled={restoring || !!restoreResult} />
        </Field>

        {restoring && (
          <div className="flex items-center gap-sm rounded-lg border border-primary/40 bg-primary/10 p-md text-on-surface">
            <CircleNotch size={18} className="animate-spin text-primary" />
            <span className="font-body-md text-body-md">Restore is in progress. Keep this window open until it finishes.</span>
          </div>
        )}

        {restoreResult?.status === "completed" && (
          <div className="flex items-center gap-sm rounded-lg border border-success/40 bg-success/10 p-md text-success">
            <CheckCircle size={18} />
            <span className="font-body-md text-body-md">Restore completed successfully.</span>
          </div>
        )}

        {restoreError && (
          <div className="rounded-lg border border-error/40 bg-error/10 p-md font-body-md text-body-md text-error">
            Restore failed: {restoreError}
          </div>
        )}

        <div className="flex items-center justify-end gap-sm border-t border-outline-variant pt-md">
          <Button variant="secondary" onClick={onClose} disabled={restoring}>
            {restoreResult ? "Close" : "Cancel"}
          </Button>
          <Button variant="danger" onClick={() => void run()} loading={restoring} disabled={!ready || !nameMatches || restoring || !!restoreResult}>
            Restore Backup
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
