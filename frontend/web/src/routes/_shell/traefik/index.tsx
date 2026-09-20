import { useEffect, useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { AlertDialog, Badge, Button, Card, InlineError, Skeleton, Textarea, useToast } from "@aether/design-system";
import { ArrowsClockwise, FileCode, FolderOpen, LockKey, Play, Trash } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { PageHeader } from "../../../components/PageHeader";
import { useDeleteTraefikFile, useRestartTraefik, useSaveTraefikFile, useTraefikFile, useTraefikFiles, useTraefikStatus } from "../../../hooks";

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KiB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MiB`;
}

function TraefikFileSystem() {
  const files = useTraefikFiles();
  const status = useTraefikStatus();
  const [selectedPath, setSelectedPath] = useState("traefik.yml");
  const [content, setContent] = useState("");
  const file = useTraefikFile(selectedPath);
  const save = useSaveTraefikFile();
  const remove = useDeleteTraefikFile();
  const restart = useRestartTraefik();
  const { add } = useToast();
  const selectedEntry = useMemo(() => files.data?.find((entry) => entry.path === selectedPath), [files.data, selectedPath]);

  useEffect(() => {
    setContent(file.data?.content ?? "");
  }, [file.data?.content]);

  const onSave = () => save.mutate({ path: selectedPath, content }, {
    onSuccess: () => add({ title: "Traefik file saved", description: "The file provider will reload this configuration automatically.", tone: "success" }),
    onError: (error) => add({ title: "Could not save Traefik file", description: error.message, tone: "error" }),
  });

  const onRestart = () => restart.mutate(undefined, {
    onSuccess: () => add({ title: "Traefik restarted", tone: "success" }),
    onError: (error) => add({ title: "Could not restart Traefik", description: error.message, tone: "error" }),
  });

  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-lg p-6 lg:p-8">
      <PageHeader
        eyebrow="Platform"
        title="Traefik File System"
        description="Manage the Aether Traefik configuration and inspect certificate state."
        actions={
          <div className="flex flex-wrap items-center gap-sm">
            <Button variant="ghost" icon={ArrowsClockwise as unknown as DesignIcon} onClick={() => { void files.refetch(); void status.refetch(); }}>
              Refresh
            </Button>
            <Button variant="primary" icon={Play as unknown as DesignIcon} onClick={onRestart} disabled={restart.isPending}>
              Restart Traefik
            </Button>
          </div>
        }
      />

      <div className="flex items-center gap-sm rounded-lg border border-status-warning/40 bg-status-warning/10 px-md py-sm text-body-sm text-on-surface">
        <LockKey size={20} className="shrink-0 text-status-warning" />
        <span>Invalid configuration can break routing. ACME private keys are protected and never displayed or editable.</span>
      </div>

      {files.error ? <InlineError title="Could not load Traefik files" message="Check the global administrator permissions and try again." onRetry={() => files.refetch()} /> : null}

      <div className="grid min-h-[620px] grid-cols-1 gap-lg xl:grid-cols-[minmax(260px,0.7fr)_minmax(0,1.8fr)]">
        <Card padding="none" variant="elevated">
          <div className="flex items-center justify-between border-b border-border px-md py-sm">
            <div>
              <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Files</h2>
              <p className="mt-1 text-body-sm text-muted-foreground">{files.data?.length ?? 0} entries</p>
            </div>
            <FolderOpen size={20} className="text-primary" />
          </div>
          <div className="max-h-[560px] overflow-y-auto p-sm">
            {files.isLoading ? <Skeleton variant="table" /> : null}
            {(files.data ?? []).map((entry) => (
              <button
                key={entry.path}
                type="button"
                onClick={() => entry.type === "file" && setSelectedPath(entry.path)}
                className={`flex w-full items-center gap-sm rounded-md px-sm py-sm text-left transition-colors ${selectedPath === entry.path ? "bg-surface-container-high text-on-surface" : "text-on-surface-variant hover:bg-surface-container"} ${entry.type === "directory" ? "cursor-default" : "cursor-pointer"}`}
              >
                {entry.protected ? <LockKey size={17} className="shrink-0 text-status-warning" /> : <FileCode size={17} className="shrink-0 text-primary" />}
                <span className="min-w-0 flex-1 truncate font-code-md text-code-md">{entry.path}</span>
                {entry.type === "file" ? <span className="text-label-sm text-muted-foreground">{formatBytes(entry.size)}</span> : null}
              </button>
            ))}
          </div>
        </Card>

        <Card padding="none" variant="elevated">
          <div className="flex flex-wrap items-center justify-between gap-sm border-b border-border px-md py-sm">
            <div className="min-w-0">
              <h2 className="truncate font-code-md text-code-md text-on-surface">{selectedPath}</h2>
              <p className="mt-1 text-body-sm text-muted-foreground">{selectedEntry?.protected ? "Protected certificate storage" : "Scoped to the Aether Traefik state directory"}</p>
            </div>
            <div className="flex items-center gap-sm">
              {file.data?.redacted ? <Badge tone="warning">Redacted</Badge> : null}
              {file.data?.editable ? <Badge tone="info">Editable</Badge> : <Badge tone="neutral">Read only</Badge>}
            </div>
          </div>
          <div className="flex h-[560px] flex-col gap-sm p-md">
            {file.error ? <InlineError title="Could not read file" message="Select another file or refresh the filesystem." onRetry={() => file.refetch()} /> : null}
            <Textarea value={content} onChange={(event) => setContent(event.target.value)} readOnly={!file.data?.editable} className="min-h-0 flex-1 resize-none font-mono text-code-sm" aria-label="Traefik file content" />
            <div className="flex flex-wrap items-center justify-between gap-sm">
              <div className="flex items-center gap-sm text-body-sm text-muted-foreground">
                {status.data ? <Badge tone={status.data.state === "running" ? "success" : "danger"} dot>{`Traefik ${status.data.state}`}</Badge> : null}
                {file.data?.redacted ? <span>Secrets are intentionally omitted from this view.</span> : null}
              </div>
              <div className="flex items-center gap-sm">
                {file.data?.editable && selectedPath !== "traefik.yml" ? (
                  <AlertDialog trigger={<Button variant="quiet" icon={Trash as unknown as DesignIcon}>Delete</Button>} title="Delete Traefik file" description={`Delete ${selectedPath}? Traefik may stop serving routes that depend on it.`} confirmLabel="Delete file" onConfirm={() => remove.mutate(selectedPath, { onSuccess: () => { setSelectedPath("traefik.yml"); add({ title: "Traefik file deleted", tone: "success" }); }, onError: (error) => add({ title: "Could not delete Traefik file", description: error.message, tone: "error" }) })} />
                ) : null}
                <Button variant="primary" onClick={onSave} disabled={!file.data?.editable || save.isPending}>Save changes</Button>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </main>
  );
}

export const Route = createFileRoute("/_shell/traefik/")({
  component: TraefikFileSystem,
});
