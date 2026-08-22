import { useEffect, useState } from "react";
import { Check, Code, NotePencil } from "@phosphor-icons/react";
import { useCreateCompose, useProjects, useSourceControlBranches, useSourceControlConnections, useSourceControlFile, useSourceControlRepositories, useStartGitHubManifest } from "../hooks";
import { ComposeEditor } from "./ComposeEditor";
import { Button, Input, Modal, Select, SelectSearch, Skeleton, useToast } from "@aether/design-system";

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
  const { add } = useToast();
  const githubConnection = connections?.find((connection) => connection.provider === "github" && connection.status === "active");
  const { data: repositories, isLoading: repositoriesLoading } = useSourceControlRepositories(githubConnection?.installation_id);
  const [projectId, setProjectId] = useState(fixedProjectId ?? "");
  const [name, setName] = useState("");
  const [content, setContent] = useState(DEFAULT_COMPOSE);
  const [sourceMode, setSourceMode] = useState<"editor" | "git">("editor");
  const [repoQuery, setRepoQuery] = useState("");
  const [repositoryID, setRepositoryID] = useState<string | undefined>();
  const [branch, setBranch] = useState("main");
  const [composePath, setComposePath] = useState("docker-compose.yml");
  const { data: branches, isLoading: branchesLoading } = useSourceControlBranches(repositoryID, githubConnection?.installation_id);
  const fileQuery = useSourceControlFile(repositoryID, githubConnection?.installation_id, composePath, branch);
  const [creating, setCreating] = useState(false);
  useEffect(() => {
    if (fileQuery.data?.content) setContent(fileQuery.data.content);
  }, [fileQuery.data?.content]);

  const filteredRepositories = (repositories ?? []).filter((repository) => !repoQuery || repository.full_name.toLowerCase().includes(repoQuery.toLowerCase()));

  const create = async () => {
    if (!projectId || !name.trim()) {
      add({ title: "Project and name are required", tone: "error" });
      return;
    }
    setCreating(true);
    try {
      await createCompose.mutateAsync({ project_id: projectId, name, content });
      onClose();
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Failed to create stack", tone: "error" });
    } finally {
      setCreating(false);
    }
  };

  return (
    <Modal open={open} onOpenChange={(value) => { if (!value) onClose(); }} title="Docker Compose · Create" description="Live validation, dependency graph and preview on the right." size="lg">
      <form onSubmit={(event) => { event.preventDefault(); void create(); }} className="space-y-lg">

        <div className="p-xl space-y-lg max-h-[70vh] overflow-y-auto sidebar-scroll">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
            <div>
              <Select label="Project" value={projectId} onValueChange={(value) => setProjectId(value ?? "")} disabled={!!fixedProjectId} options={[{ label: "Select...", value: "" }, ...(projects ?? []).map((p) => ({ label: p.name, value: p.id }))]} />
            </div>
            <div>
              <Input label="Stack name" placeholder="my-stack" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-md">
            {[
              { id: "editor" as const, label: "Editor", icon: NotePencil, description: "Write or paste your Compose YAML." },
              { id: "git" as const, label: "Git Provider", icon: Code, description: "Load docker-compose.yml from a connected repository." },
            ].map((option) => (
              <Button key={option.id} type="button" variant={sourceMode === option.id ? "secondary" : "outline"} onClick={() => setSourceMode(option.id)} className="h-auto min-h-24 justify-start p-md text-left">
                <option.icon size={18} className={sourceMode === option.id ? "text-primary" : "text-on-surface-variant"} />
                <span className="flex-1">
                  <span className="flex items-center gap-xs font-body-md text-body-md font-semibold text-on-surface">{option.label}{sourceMode === option.id ? <Check size={16} className="ml-auto text-primary" /> : null}</span>
                  <span className="block mt-xs font-body-sm text-body-sm text-on-surface-variant">{option.description}</span>
                </span>
              </Button>
            ))}
          </div>
          {sourceMode === "git" && (
            <div className="flex flex-col gap-md bg-surface-container-low border border-outline-variant rounded-xl p-md">
              {!githubConnection ? (
                <div className="flex items-center justify-between gap-md">
                  <p className="font-body-sm text-body-sm text-on-surface-variant">Connect a GitHub account to load a Compose file.</p>
                  <Button type="button" variant="ghost" disabled={startGitHubManifest.isPending} onClick={async () => {
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
                      add({ title: error instanceof Error ? error.message : "GitHub connection failed", tone: "error" });
                    }
                  }}>Connect GitHub</Button>
                </div>
              ) : (
                <>
                  <SelectSearch label="Repository" value={repositoryID ?? null} onValueChange={(value) => { const repository = (repositories ?? []).find((item) => item.id === value); setRepositoryID(value ?? undefined); setBranch(repository?.default_branch || "main"); setRepoQuery(repository?.full_name || ""); }} options={filteredRepositories.map((repository) => ({ label: repository.full_name, value: repository.id }))} placeholder="Search repositories" />
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-md">
                    {branchesLoading ? <div><span className="mb-xs block text-label-md text-on-surface">Branch</span><Skeleton variant="table" aria-label="Loading branches" /></div> : <Select label="Branch" value={branch} onValueChange={(value) => setBranch(value ?? branch)} options={branches?.length ? branches.map((item) => ({ label: item.name, value: item.name })) : [{ label: branch, value: branch }]} />}
                    <Input label="Compose file" value={composePath} onChange={(event) => setComposePath(event.target.value)} placeholder="docker-compose.yml" />
                  </div>
                  {fileQuery.isFetching && <div className="space-y-sm" aria-label="Loading Compose file"><Skeleton variant="text" /><Skeleton variant="text" className="w-2/3" /></div>}
                  {fileQuery.isError && <p className="font-body-sm text-body-sm text-error">Could not load that file from the repository.</p>}
                </>
              )}
            </div>
          )}
          <ComposeEditor value={content} onChange={setContent} />
        </div>

        <div className="flex justify-between border-t border-outline-variant pt-lg">
          <Button type="button" variant="ghost" onClick={onClose}>Cancel</Button>
          <Button type="submit" loading={creating}>
            Create stack
          </Button>
        </div>
      </form>
    </Modal>
  );
}
