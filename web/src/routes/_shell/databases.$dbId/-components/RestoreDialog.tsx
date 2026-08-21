import { useState } from "react";
import type { BackupJob } from "../../../../api/types";
import { useDatabaseBackupPreflight, useDatabaseBackupRestore } from "../../../../hooks";
import { Button, Field, Input, Modal } from "../../../../components/ui";

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
  const restore = useDatabaseBackupRestore(dbId);
  const { data: preflight } = useDatabaseBackupPreflight(dbId, backup.id, true);
  const [restoring, setRestoring] = useState(false);

  const ready = preflight?.ready ?? false;
  const nameMatches = !!dbName && confirm.trim() === dbName;

  const run = async () => {
    setRestoring(true);
    try {
      await restore.mutateAsync(backup.id);
      onClose();
    } finally {
      setRestoring(false);
    }
  };

  return (
    <Modal open onClose={onClose} title="Restore backup" wide>
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
                <span className={`material-symbols-outlined text-[18px] ${c.ok ? "text-success" : "text-error"}`}>
                  {c.ok ? "check_circle" : "cancel"}
                </span>
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
          <Input value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder={dbName} />
        </Field>

        <div className="flex items-center justify-end gap-sm border-t border-outline-variant pt-md">
          <Button variant="subtle" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="danger" onClick={() => void run()} loading={restoring} disabled={!ready || !nameMatches}>
            Restore Backup
          </Button>
        </div>
      </div>
    </Modal>
  );
}