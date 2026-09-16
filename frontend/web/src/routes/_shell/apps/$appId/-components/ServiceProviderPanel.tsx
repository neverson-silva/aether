import { Code, Link } from "@phosphor-icons/react";
import { Button, Checkbox, Input, NativeSelect, Switch } from "@aether/design-system";
import type { App } from "@/api/types";
import type { ServiceSource, SourceControlRepository } from "@/hooks/use-source-control";

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
    <section className="mb-8 space-y-8 rounded-xl border border-outline-variant bg-surface-container-lowest p-8">
      <div className="flex items-center justify-between"><div><h3 className="font-headline-sm text-headline-sm text-on-surface">Provider</h3><p className="text-body-sm text-on-surface-variant">Source of this service's code or image</p></div><Link size={18} className="text-muted-foreground" /></div>
      <div className="flex gap-6 border-b border-outline-variant pb-4">
        {["Git", "Image", "Upload"].map((label) => <button key={label} type="button" className={`flex items-center gap-2 pb-4 text-body-sm font-medium transition-colors ${((label === "Git" && app.source_type === "git") || (label === "Image" && app.source_type === "image")) ? "-mb-[17px] border-b-2 border-primary font-bold text-primary" : "text-on-surface-variant hover:text-on-surface"}`}><Code size={18} />{label}</button>)}
      </div>
      <div className="grid grid-cols-1 gap-6">
        <div className="space-y-2"><label className="block font-label-caps text-label-caps text-on-surface-variant">{app.source_type === "git" ? "Repository" : "Image"}</label>{editing && source ? <NativeSelect value={repositoryId} onChange={(event) => onRepositoryChange(event.target.value)} options={[{ label: source.repository_full_name || "Current repository", value: source.repository_id }, ...(repositories ?? []).filter((repository) => repository.id !== source.repository_id).map((repository) => ({ label: repository.full_name, value: repository.id }))]} /> : <ValueBox value={app.source_type === "git" ? source?.repository_full_name || app.git_url || "—" : app.image} />}</div>
        <div className="grid grid-cols-2 gap-6"><div className="space-y-2"><label className="block font-label-caps text-label-caps text-on-surface-variant">Branch</label>{editing && source ? <Input value={branch} onChange={(event) => onBranchChange(event.target.value)} /> : <ValueBox value={source?.branch || app.git_branch || "main"} />}</div><div className="space-y-2"><label className="block font-label-caps text-label-caps text-on-surface-variant">Port</label><ValueBox value={`:${app.port}`} /></div></div>
      </div>
      {source ? <div className="grid grid-cols-1 gap-4 border-t border-outline-variant pt-4 md:grid-cols-3"><div className="space-y-2"><span className="block font-label-caps text-label-caps text-on-surface-variant">Root directory</span>{editing ? <Input value={rootDirectory} onChange={(event) => onRootDirectoryChange(event.target.value)} /> : <span className="font-code-md text-code-md text-on-surface">{source.root_directory || "/"}</span>}</div><div className="space-y-2"><span className="block font-label-caps text-label-caps text-on-surface-variant">Watch paths</span>{editing ? <Input value={watchPaths} onChange={(event) => onWatchPathsChange(event.target.value)} /> : <span className="font-code-md text-code-md text-on-surface">{source.watch_paths.length || "All service files"}</span>}</div><div className="space-y-2"><span className="block font-label-caps text-label-caps text-on-surface-variant">Root files</span>{editing ? <Checkbox label="Include root files" checked={watchRootFiles} onCheckedChange={(checked) => onWatchRootFilesChange(checked === true)} /> : <span className="font-code-md text-code-md text-on-surface">{source.watch_root_files ? "Included" : "Ignored"}</span>}</div></div> : null}
      {editing && source ? <div className="flex flex-wrap items-center justify-between gap-4 border-t border-outline-variant pt-4"><div className="flex items-center gap-3"><span className="text-body-sm font-medium text-foreground">Automatic deploys</span><Switch ariaLabel="Automatic deploys" checked={autoDeploy} onCheckedChange={onAutoDeployChange} /></div><div className="flex gap-3"><Button variant="ghost" onClick={onCancel}>Cancel</Button><Button loading={saving} onClick={onSave}>Save changes</Button></div></div> : <div className="flex justify-end pt-4"><Button onClick={onEdit}>Edit</Button></div>}
    </section>
  );
}

function ValueBox({ value }: { value: string }) {
  return <div className="flex w-full items-center justify-between rounded border border-outline-variant bg-surface-container p-3"><span className="truncate font-code-md text-code-md text-on-surface">{value}</span></div>;
}
