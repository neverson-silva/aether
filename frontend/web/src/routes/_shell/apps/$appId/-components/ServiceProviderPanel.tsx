import { Code, Link } from "@phosphor-icons/react";
import {
  Button,
  Checkbox,
  Input,
  NativeSelect,
  Switch,
} from "@aether/design-system";
import type { App } from "@/api/types";
import type {
  ServiceSource,
  SourceControlRepository,
} from "@/hooks/use-source-control";

export function ServiceProviderPanel({
  app,
  source,
  repositories,
  editing,
  repositoryId,
  branch,
  rootDirectory,
  watchPaths,
  watchRootFiles,
  autoDeploy,
  saving,
  onRepositoryChange,
  onBranchChange,
  onRootDirectoryChange,
  onWatchPathsChange,
  onWatchRootFilesChange,
  onAutoDeployChange,
  onEdit,
  onCancel,
  onSave,
}: {
  app: App;
  source?: ServiceSource;
  repositories?: SourceControlRepository[];
  editing: boolean;
  repositoryId: string;
  branch: string;
  rootDirectory: string;
  watchPaths: string;
  watchRootFiles: boolean;
  autoDeploy: boolean;
  saving: boolean;
  onRepositoryChange: (value: string) => void;
  onBranchChange: (value: string) => void;
  onRootDirectoryChange: (value: string) => void;
  onWatchPathsChange: (value: string) => void;
  onWatchRootFilesChange: (value: boolean) => void;
  onAutoDeployChange: (value: boolean) => void;
  onEdit: () => void;
  onCancel: () => void;
  onSave: () => void;
}) {
  return (
    <section className="mb-lg space-y-lg overflow-hidden rounded-2xl border border-outline-variant bg-surface-container shadow-md">
      <div className="flex flex-wrap items-start justify-between gap-md border-b border-outline-variant px-lg py-lg">
        <div>
          <p className="font-label-caps text-label-caps uppercase text-primary">
            Source control
          </p>
          <h3 className="mt-xs font-headline-sm text-headline-sm text-on-surface">
            Provider
          </h3>
          <p className="mt-xs text-body-sm text-on-surface-variant">
            Source of this service's code or image
          </p>
        </div>
        <div className="flex items-center gap-sm rounded-full border border-status-success/30 bg-status-success-container/20 px-md py-xs font-label-caps text-label-caps uppercase text-status-success">
          <span className="size-1.5 rounded-full bg-status-success" />
          Connected
          <Link size={16} aria-hidden="true" />
        </div>
      </div>
      <div className="px-lg">
        <div className="flex flex-wrap gap-1 rounded-xl border border-outline-variant bg-surface-container-lowest/60 p-1">
          {["Git", "Image", "Upload"].map((label) => (
            <button
              key={label}
              type="button"
              className={`flex items-center gap-2 rounded-lg px-md py-sm text-body-sm font-medium outline-none transition-[background-color,color,transform] duration-200 active:scale-[0.98] focus-visible:ring-2 focus-visible:ring-ring ${(label === "Git" && app.source_type === "git") || (label === "Image" && app.source_type === "image") ? "bg-surface-container-high font-bold text-primary shadow-sm" : "text-on-surface-variant hover:bg-surface-container hover:text-on-surface"}`}
            >
              <Code size={18} />
              {label}
            </button>
          ))}
        </div>
      </div>
      <div className="grid grid-cols-1 gap-lg px-lg sm:grid-cols-2">
        <div className="space-y-sm sm:col-span-2">
          <label className="block font-label-caps text-label-caps text-on-surface-variant">
            {app.source_type === "git" ? "Repository" : "Image"}
          </label>
          {editing && source ? (
            <NativeSelect
              value={repositoryId}
              onChange={(event) => onRepositoryChange(event.target.value)}
              options={[
                {
                  label: source.repository_full_name || "Current repository",
                  value: source.repository_id,
                },
                ...(repositories ?? [])
                  .filter(
                    (repository) => repository.id !== source.repository_id,
                  )
                  .map((repository) => ({
                    label: repository.full_name,
                    value: repository.id,
                  })),
              ]}
            />
          ) : (
            <ValueBox
              value={
                app.source_type === "git"
                  ? source?.repository_full_name || app.git_url || "—"
                  : app.image
              }
            />
          )}
        </div>
        <div className="space-y-sm">
          <label className="block font-label-caps text-label-caps text-on-surface-variant">
            Branch
          </label>
          {editing && source ? (
            <Input
              value={branch}
              onChange={(event) => onBranchChange(event.target.value)}
            />
          ) : (
            <ValueBox value={source?.branch || app.git_branch || "main"} />
          )}
        </div>
        <div className="space-y-sm">
          <label className="block font-label-caps text-label-caps text-on-surface-variant">
            Port
          </label>
          <ValueBox value={`:${app.port}`} />
        </div>
      </div>
      {source ? (
        <div className="mx-lg grid grid-cols-1 gap-md border-t border-outline-variant py-lg md:grid-cols-3">
          <div className="rounded-xl bg-surface-container-lowest/60 p-md">
            <span className="block font-label-caps text-label-caps text-on-surface-variant">
              Root directory
            </span>
            {editing ? (
              <Input
                value={rootDirectory}
                onChange={(event) => onRootDirectoryChange(event.target.value)}
              />
            ) : (
              <span className="mt-sm block font-code-md text-code-md text-on-surface">
                {source.root_directory || "/"}
              </span>
            )}
          </div>
          <div className="rounded-xl bg-surface-container-lowest/60 p-md">
            <span className="block font-label-caps text-label-caps text-on-surface-variant">
              Watch paths
            </span>
            {editing ? (
              <Input
                value={watchPaths}
                onChange={(event) => onWatchPathsChange(event.target.value)}
              />
            ) : (
              <span className="mt-sm block font-code-md text-code-md text-on-surface">
                {source.watch_paths.length || "All service files"}
              </span>
            )}
          </div>
          <div className="rounded-xl bg-surface-container-lowest/60 p-md">
            <span className="block font-label-caps text-label-caps text-on-surface-variant">
              Root files
            </span>
            {editing ? (
              <Checkbox
                label="Include root files"
                checked={watchRootFiles}
                onCheckedChange={(checked) =>
                  onWatchRootFilesChange(checked === true)
                }
              />
            ) : (
              <span className="mt-sm block font-code-md text-code-md text-on-surface">
                {source.watch_root_files ? "Included" : "Ignored"}
              </span>
            )}
          </div>
        </div>
      ) : null}
      {editing && source ? (
        <div className="flex flex-wrap items-center justify-between gap-md border-t border-outline-variant px-lg py-lg">
          <div className="flex items-center gap-md">
            <span className="text-body-sm font-medium text-foreground">
              Automatic deploys
            </span>
            <Switch
              ariaLabel="Automatic deploys"
              checked={autoDeploy}
              onCheckedChange={onAutoDeployChange}
            />
          </div>
          <div className="flex gap-sm">
            <Button variant="ghost" onClick={onCancel}>
              Cancel
            </Button>
            <Button loading={saving} onClick={onSave}>
              Save changes
            </Button>
          </div>
        </div>
      ) : (
        <div className="flex justify-end px-lg pb-lg">
          <Button onClick={onEdit}>Edit</Button>
        </div>
      )}
    </section>
  );
}

function ValueBox({ value }: { value: string }) {
  return (
    <div className="flex w-full items-center justify-between rounded-xl border border-outline-variant bg-surface-container-lowest px-md py-sm">
      <span className="truncate font-code-md text-code-md text-on-surface">
        {value}
      </span>
    </div>
  );
}
