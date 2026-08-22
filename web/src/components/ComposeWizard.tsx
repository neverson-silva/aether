import { useEffect, useRef, useState } from "react";
import { useCreateCompose, useProjects, useSourceControlBranches, useSourceControlConnections, useSourceControlFile, useSourceControlRepositories, useStartGitHubManifest } from "../hooks";
import { useOverlayGate } from "./OverlayManager";
import { ComposeEditor } from "./ComposeEditor";
import { Button, Input, Select, useToast } from "./ui";

const DEFAULT_COMPOSE = `services:
  app:
    image: nginx:alpine
    ports:
      - "80:80"
    restart: unless-stopped`;

export function ComposeWizard({ open, onClose, fixedProjectId }: { open: boolean; onClose: () => void; fixedProjectId?: string }) {
  const { data: projects } = useProjects();
  const createCompose = useCreateCompose();
  const { data: connections } = useSourceControlConnections();
  const startGitHubManifest = useStartGitHubManifest();
  const { toast } = useToast();
  const githubConnection = connections?.find((connection) => connection.provider === "github" && connection.status === "active");
  const { data: repositories, isLoading: repositoriesLoading } = useSourceControlRepositories(githubConnection?.installation_id);
  const [projectId, setProjectId] = useState(fixedProjectId ?? "");
  const [name, setName] = useState("");
  const [content, setContent] = useState(DEFAULT_COMPOSE);
  const [sourceMode, setSourceMode] = useState<"editor" | "git">("editor");
  const [repoQuery, setRepoQuery] = useState("");
  const [repoPickerOpen, setRepoPickerOpen] = useState(false);
  const repoPickerRef = useRef<HTMLDivElement>(null);
  const [repositoryID, setRepositoryID] = useState<string | undefined>();
  const [branch, setBranch] = useState("main");
  const [composePath, setComposePath] = useState("docker-compose.yml");
  const { data: branches, isLoading: branchesLoading } = useSourceControlBranches(repositoryID, githubConnection?.installation_id);
  const fileQuery = useSourceControlFile(repositoryID, githubConnection?.installation_id, composePath, branch);
  const [creating, setCreating] = useState(false);
  const { mounted, closing, close } = useOverlayGate("compose-wizard", open, onClose);

  useEffect(() => {
    if (fileQuery.data?.content) setContent(fileQuery.data.content);
  }, [fileQuery.data?.content]);

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (!repoPickerRef.current?.contains(event.target as Node)) setRepoPickerOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  const filteredRepositories = (repositories ?? []).filter((repository) => !repoQuery || repository.full_name.toLowerCase().includes(repoQuery.toLowerCase()));

  if (!mounted) return null;

  const create = async () => {
    if (!projectId || !name.trim()) {
      toast("Project and name are required", "error");
      return;
    }
    setCreating(true);
    try {
      await createCompose.mutateAsync({ project_id: projectId, name, content });
      onClose();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create stack", "error");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className={`fixed inset-0 z-[80] flex items-center justify-center bg-black/60 p-4 ${closing ? "animate-fade-out" : "animate-fade-in"}`} onClick={() => close()}>
      <div role="dialog" aria-modal="true" aria-label="Compose wizard" onClick={(e) => e.stopPropagation()} className="w-full max-w-4xl bg-surface-modal border border-outline-variant rounded-2xl shadow-xl animate-modal-pop overflow-hidden">
        <div className="flex items-center justify-between px-xl pt-xl pb-md border-b border-outline-variant">
          <div>
            <h2 className="font-headline-sm text-headline-sm text-on-surface">🐳 Docker Compose · Create</h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant">Live validation, dependency graph and preview on the right.</p>
          </div>
          <button onClick={() => close()} className="material-symbols-outlined text-on-surface-variant hover:text-on-surface transition-colors">close</button>
        </div>

        <div className="p-xl space-y-lg max-h-[70vh] overflow-y-auto sidebar-scroll">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
            <div>
              <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Project</p>
              <Select value={projectId} onChange={(e) => setProjectId(e.target.value)} disabled={!!fixedProjectId}>
                <option value="">Select...</option>
                {(projects ?? []).map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </Select>
            </div>
            <div>
              <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Stack name</p>
              <Input icon="label" placeholder="my-stack" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-md">
            {[
              { id: "editor" as const, label: "Editor", icon: "edit_note", description: "Write or paste your Compose YAML." },
              { id: "git" as const, label: "Git Provider", icon: "code", description: "Load docker-compose.yml from a connected repository." },
            ].map((option) => (
              <button key={option.id} type="button" onClick={() => setSourceMode(option.id)} className={`flex items-start gap-sm p-md rounded-xl border text-left ${sourceMode === option.id ? "border-primary bg-primary/10" : "border-outline-variant bg-surface-container-low"}`}>
                <span className={`material-symbols-outlined ${sourceMode === option.id ? "text-primary" : "text-on-surface-variant"}`}>{option.icon}</span>
                <span className="flex-1">
                  <span className="flex items-center gap-xs font-body-md text-body-md font-semibold text-on-surface">{option.label}<span className="material-symbols-outlined ml-auto text-[16px]">{sourceMode === option.id ? "check_box" : "check_box_outline_blank"}</span></span>
                  <span className="block mt-xs font-body-sm text-body-sm text-on-surface-variant">{option.description}</span>
                </span>
              </button>
            ))}
          </div>
          {sourceMode === "git" && (
            <div className="flex flex-col gap-md bg-surface-container-low border border-outline-variant rounded-xl p-md">
              {!githubConnection ? (
                <div className="flex items-center justify-between gap-md">
                  <p className="font-body-sm text-body-sm text-on-surface-variant">Connect a GitHub account to load a Compose file.</p>
                  <button type="button" disabled={startGitHubManifest.isPending} onClick={async () => {
                    try {
                      const manifest = await startGitHubManifest.mutateAsync({ return_url: `${window.location.pathname}${window.location.search}` });
                      const form = document.createElement("form");
                      form.method = "POST";
                      form.action = `${manifest.url}?state=${encodeURIComponent(manifest.state)}`;
                      const input = document.createElement("input");
                      input.type = "hidden";
                      input.name = "manifest";
                      input.value = manifest.manifest;
                      form.appendChild(input);
                      document.body.appendChild(form);
                      form.submit();
                    } catch (error) {
                      toast(error instanceof Error ? error.message : "GitHub connection failed", "error");
                    }
                  }} className="text-primary font-body-sm text-body-sm">Connect GitHub</button>
                </div>
              ) : (
                <>
                  <div ref={repoPickerRef} onMouseDown={() => setRepoPickerOpen(true)} className="group relative">
                    <Input value={repoQuery} onFocus={() => setRepoPickerOpen(true)} onClick={() => setRepoPickerOpen(true)} onChange={(event) => { setRepoQuery(event.target.value); setRepoPickerOpen(true); }} placeholder="Search repositories" />
                  <div className="relative mt-xs hidden max-h-40 overflow-y-auto flex-col gap-xs rounded-lg border border-outline-variant bg-surface-container-high p-xs shadow-xl group-focus-within:flex">
                    {repositoriesLoading && <p className="font-body-sm text-body-sm text-on-surface-variant">Loading repositories...</p>}
                    {filteredRepositories.map((repository) => <button key={repository.id} type="button" onMouseDown={() => { setRepositoryID(repository.id); setBranch(repository.default_branch || "main"); setRepoQuery(repository.full_name); setRepoPickerOpen(false); }} className={`text-left p-sm rounded border ${repositoryID === repository.id ? "border-primary bg-primary/10" : "border-outline-variant"}`}>{repository.full_name}</button>)}
                  </div>
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-md">
                    <Select value={branch} onChange={(event) => setBranch(event.target.value)} disabled={branchesLoading}>
                      {branches?.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}
                      {!branches?.length && <option value={branch}>{branchesLoading ? "Loading branches..." : branch}</option>}
                    </Select>
                    <Input value={composePath} onChange={(event) => setComposePath(event.target.value)} placeholder="docker-compose.yml" />
                  </div>
                  {fileQuery.isFetching && <p className="font-body-sm text-body-sm text-on-surface-variant">Loading Compose file...</p>}
                  {fileQuery.isError && <p className="font-body-sm text-body-sm text-error">Could not load that file from the repository.</p>}
                </>
              )}
            </div>
          )}
          <ComposeEditor value={content} onChange={setContent} />
        </div>

        <div className="flex justify-between px-xl py-lg border-t border-outline-variant">
          <Button variant="ghost" onClick={() => close()}>Cancel</Button>
          <Button onClick={create} disabled={creating}>
            {creating ? "Creating..." : "Create stack"}
          </Button>
        </div>
      </div>
    </div>
  );
}
