import { useRef, useState } from "react";
import type { RestoreJob } from "../../../../api/types";
import { useDatabaseUploadRestore, useRestoreJob } from "../../../../hooks";
import { Button, Dialog, Field, Input, useToast } from "@aether/design-system";
import { CheckCircle, CircleNotch, FileArrowUp, XCircle } from "@phosphor-icons/react";
import { useQueryClient } from "@tanstack/react-query";

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MiB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GiB`;
}

type Phase = "pick" | "uploading" | "ready" | "restoring" | "terminal";

const FORMAT_LABELS: Record<string, string> = {
  dump: "PostgreSQL Custom Dump",
  tar: "PostgreSQL Tar Dump",
  sql: "SQL Dump",
  "sql.gz": "Gzip Compressed SQL Dump",
  gzip: "Gzip Compressed Archive",
  bak: "SQL Server Backup",
  dmp: "Oracle Data Pump",
};

function formatLabel(format: string): string {
  return FORMAT_LABELS[format] ?? (format.toUpperCase() || format);
}

export function UploadRestoreDialog({
  dbId,
  dbName,
  onClose,
}: {
  dbId: string;
  dbName?: string;
  onClose: () => void;
}) {
  const { add } = useToast();
  const queryClient = useQueryClient();
  const inputRef = useRef<HTMLInputElement>(null);
  const { createRestore, uploadRestore, validateRestore, startRestore, cancelRestore } = useDatabaseUploadRestore(dbId);
  const [phase, setPhase] = useState<Phase>("pick");
  const [file, setFile] = useState<File | null>(null);
  const [progress, setProgress] = useState<{ loaded: number; total: number } | null>(null);
  const [selected, setSelected] = useState<RestoreJob | null>(null);
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const finalRestore = useRestoreJob(dbId, selected?.id ?? null);
  const job: RestoreJob | null =
    (finalRestore.data && ["uploading", "validating", "ready", "queued"].includes(finalRestore.data.status)
      ? selected
      : finalRestore.data) ?? selected;

  const pick = (f: File) => {
    setError(null);
    setFile(f);
    setBusy(true);
    createRestore.mutate(f.name, {
      onSuccess: (created) => {
        setSelected(created);
        setPhase("uploading");
        uploadRestore.mutate(
          { restoreId: created.id, file: f, onProgress: (loaded, total) => setProgress({ loaded, total }) },
          {
            onSuccess: (done) => {
              setSelected(done);
              setPhase(done.status === "ready" ? "ready" : "terminal");
            },
            onError: (e) => {
              setError(e instanceof Error ? e.message : "Upload failed");
              setPhase("terminal");
            },
            onSettled: () => setBusy(false),
          }
        );
      },
      onError: (e) => {
        setError(e instanceof Error ? e.message : "Could not start restore");
        setPhase("terminal");
        setBusy(false);
      },
    });
  };

  const validate = () => {
    if (!selected) return;
    setBusy(true);
    validateRestore.mutate(selected.id, {
      onSuccess: (done) => {
        setSelected(done);
        setPhase("ready");
      },
      onError: (e) => setError(e instanceof Error ? e.message : "Validation failed"),
      onSettled: () => setBusy(false),
    });
  };

  const start = () => {
    if (!selected || confirm !== dbName) return;
    setBusy(true);
    startRestore.mutate(selected.id, {
      onSuccess: (job) => {
        add({ title: "Restore started", tone: "info" });
        setSelected(job);
        setPhase("restoring");
      },
      onError: (e) => setError(e instanceof Error ? e.message : "Could not start restore"),
      onSettled: () => setBusy(false),
    });
  };

  const cancel = () => {
    if (!selected) return;
    setBusy(true);
    cancelRestore.mutate(selected.id, {
      onSuccess: () => {
        add({ title: "Restore cancelled", tone: "info" });
        queryClient.invalidateQueries({ queryKey: ["database-restore-jobs", dbId] });
        onClose();
      },
      onError: (e) => setError(e instanceof Error ? e.message : "Could not cancel"),
      onSettled: () => setBusy(false),
    });
  };

  const running = job && ["queued", "uploading", "validating", "ready", "preparing", "downloading", "restoring"].includes(job.status);
  const done = job && ["completed", "failed", "cancelled"].includes(job.status);

  const drop = (e: React.DragEvent) => {
    e.preventDefault();
    const f = e.dataTransfer.files?.[0];
    if (f) pick(f);
  };

  return (
    <Dialog open trigger={<span />} onOpenChange={(open) => { if (!open && !busy) onClose(); }} title="Import backup from file">
      <div className="space-y-lg">
        {error && (
          <div className="rounded-lg border border-error/40 bg-error/10 p-md font-body-md text-body-md text-error">
            {error}
          </div>
        )}

        {phase === "pick" && (
          <div
            className="rounded-xl border border-dashed border-outline-variant p-xl text-center cursor-pointer hover:border-primary/60 transition-colors"
            onDrop={drop}
            onDragOver={(e) => e.preventDefault()}
            onClick={() => inputRef.current?.click()}
            role="button"
            aria-label="Upload restore file"
          >
            <input
              ref={inputRef}
              type="file"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) pick(f);
              }}
            />
            <FileArrowUp size={40} className="mx-auto text-on-surface-variant/60" />
            <p className="font-body-md text-body-md text-on-surface mt-md">Drop a dump file here or click to browse</p>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-0.5">
              Supported: Postgres (.dump/.backup/.sql/.sql.gz), MySQL/MariaDB (.sql/.sql.gz), SQL Server (.bak), Oracle (.dmp)
            </p>
          </div>
        )}

        {phase === "uploading" && file && (
          <div className="space-y-sm">
            <div className="flex items-center justify-between">
              <span className="font-body-md text-body-md text-on-surface">{file.name}</span>
              <span className="font-code-md text-code-md text-on-surface-variant">
                {progress ? `${formatBytes(progress.loaded)} / ${progress.total ? formatBytes(progress.total) : "?"}` : "…"}
              </span>
            </div>
            <div className="h-2 rounded-full bg-surface-container-high overflow-hidden">
              <div
                className="h-full bg-primary transition-[width]"
                style={{ width: progress?.total ? `${Math.min(100, (progress.loaded / progress.total) * 100)}%` : "20%" }}
              />
            </div>
            {busy && <p className="font-body-sm text-body-sm text-on-surface-variant">Uploading… keep this window open.</p>}
          </div>
        )}

        {phase === "ready" && job && (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-md">
            <div className="px-md py-sm rounded border border-outline-variant/60">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">File</span>
              <span className="font-code-md text-code-md text-on-surface break-all">{job.source_filename}</span>
            </div>
            <div className="px-md py-sm rounded border border-outline-variant/60">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Size</span>
              <span className="font-code-md text-code-md text-on-surface">{formatBytes(job.source_size)}</span>
            </div>
            <div className="px-md py-sm rounded border border-outline-variant/60">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Detected format</span>
              <span className="font-code-md text-code-md text-on-surface">{formatLabel(job.source_format)}</span>
            </div>
            <div className="px-md py-sm rounded border border-outline-variant/60">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Target</span>
              <span className="font-code-md text-code-md text-on-surface">{dbName ?? dbId}</span>
            </div>
          </div>
        )}

        {phase === "ready" && job && (
          <>
            <div className="p-md rounded-lg border border-warning/40 bg-warning/10">
              <p className="font-body-md text-body-md text-on-surface font-medium">Restore database?</p>
              <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">
                This will replace the current data in {dbName ?? "the database"} with the contents of {job.source_filename}.
                This action may cause temporary downtime. Type the database name to continue.
              </p>
            </div>
            <Field label={`Type "${dbName ?? "database"}" to confirm`}>
              <Input value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder={dbName} disabled={busy} />
            </Field>
          </>
        )}

        {phase === "restoring" && running && (
          <div className="flex items-center gap-sm rounded-lg border border-primary/40 bg-primary/10 p-md text-on-surface">
            <CircleNotch size={18} className="animate-spin text-primary" />
            <span className="font-body-md text-body-md">Restore in progress: {job?.status}. Keep this window open.</span>
          </div>
        )}

        {done && job && (
          <div className={`flex items-center gap-sm rounded-lg border p-md ${job.status === "completed" ? "border-success/40 bg-success/10 text-success" : "border-error/40 bg-error/10 text-error"}`}>
            {job.status === "completed" ? <CheckCircle size={18} /> : <XCircle size={18} />}
            <span className="font-body-md text-body-md">
              {job.status === "completed" ? "Restore completed successfully." : job.error_message || `Restore ${job.status}.`}
            </span>
          </div>
        )}

        <div className="flex items-center justify-end gap-sm border-t border-outline-variant pt-md">
          {phase === "ready" && job?.status === "ready" && !busy && (
            <Button variant="secondary" onClick={cancel}>Cancel</Button>
          )}
          <Button variant="secondary" onClick={onClose} disabled={busy}>
            {done ? "Close" : "Close"}
          </Button>
          {phase === "ready" && job?.status === "ready" && (
            <Button variant="danger" onClick={start} loading={busy} disabled={confirm !== dbName}>
              Restore backup
            </Button>
          )}
        </div>
      </div>
    </Dialog>
  );
}
