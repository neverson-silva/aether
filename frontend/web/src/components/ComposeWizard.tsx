import { useState } from "react";
import { Check, Code, NotePencil } from "@phosphor-icons/react";
import { apiGet, apiPut } from "../api/client";
import { useCreateCompose, useProjects, useSourceControlBranches, useSourceControlConnections, useSourceControlRepositories, useStartGitHubManifest } from "../hooks";
import { ComposeEditor } from "./ComposeEditor";
import { Button, Input, Modal, Select, SelectSearch, Skeleton, useToast } from "@aether/design-system";

const DEFAULT_COMPOSE = `services:
  app:
    image: nginx:alpine
    ports:
      - "80:80"
    restart: unless-stopped`;

export function ComposeWizard({ open, onClose, fixedProjectId, fixedEnvironmentId, onCreated }: { open: boolean; onClose: () => void; fixedProjectId?: string; fixedEnvironmentId?: string; onCreated?: (serviceId: string) => void }) {
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
  const { data: branches, isLoading: branchesLoading, isError: branchesError } = useSourceControlBranches(repositoryID, githubConnection?.installation_id);
  const [creating, setCreating] = useState(false);

  const filteredRepositories = (repositories ?? []).filter((repository) => !repoQuery || repository.full_name.toLowerCase().includes(repoQuery.toLowerCase()));

  const create = async () => {
    if (!projectId || !name.trim()) {
      add({ title: "Project and name are required", tone: "error" });
      return;
    }
    setCreating(true);
    try {
      let compose = content;
      if (sourceMode === "git") {
        if (!repositoryID || !githubConnection?.installation_id || !branch.trim() || !composePath.trim()) {
          add({ title: "Select a repository, branch and Compose file", tone: "error" });
          return;
        }
        try {
          const file = await apiGet<{ content: string }>(`/api/v1/source-control/github/repositories/${encodeURIComponent(repositoryID)}/file?installation_id=${encodeURIComponent(githubConnection.installation_id)}&path=${encodeURIComponent(composePath)}&ref=${encodeURIComponent(branch)}`);
          compose = file.content;
        } catch {
          add({ title: "Could not load the selected Compose file", tone: "error" });
          return;
        }
      }
      if (!compose.trim()) {
        add({ title: "The Compose file is empty", tone: "error" });
        return;
      }
      const created = await createCompose.mutateAsync({ project_id: projectId, environment_id: fixedEnvironmentId, name, compose });
      let service: { id: string; spec_id?: string; name: string; kind: string } | undefined;
      try {
        const services = await apiGet<Array<{ id: string; spec_id?: string; name: string; kind: string }>>(`/api/v1/services?project_id=${encodeURIComponent(projectId)}`);
        service = services.find((item) => item.spec_id === created.id || (item.name === name.trim() && item.kind === "compose"));
      } catch {
        service = undefined;
      }
      if (sourceMode === "git" && repositoryID && githubConnection && service) {
        const repository = repositories?.find((item) => item.id === repositoryID);
        if (service && repository) {
          try {
            await apiPut(`/api/v1/services/${service.id}/source`, {
              connection_id: githubConnection.id,
              repository_id: repository.id,
              repository_owner: repository.owner,
              repository_name: repository.name,
              repository_full_name: repository.full_name,
              default_branch: repository.default_branch,
              branch,
              auto_deploy: false,
              root_directory: ".",
              environment_template_path: ".env.example",
              watch_paths: [],
              ignore_paths: [],
              watch_root_files: false,
            });
          } catch {
            add({ title: "Stack created, but Git settings could not be saved", tone: "warning" });
          }
        }
      }
      add({ title: "Stack created", tone: "success" });
      if (service && onCreated) onCreated(service.id);
      else onClose();
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
              {fixedProjectId ? <Input label="Project" value={projects?.find((project) => project.id === fixedProjectId)?.name ?? "Loading project..."} disabled /> : <Select label="Project" value={projectId} onValueChange={(value) => setProjectId(value ?? "")} options={[{ label: "Select...", value: "" }, ...(projects ?? []).map((p) => ({ label: p.name, value: p.id }))]} />}
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
                    {branchesLoading ? <div><span className="mb-xs block text-label-md text-on-surface">Branch</span><Skeleton variant="table" aria-label="Loading branches" /></div> : branches?.length ? <SelectSearch label="Branch" value={branch} onValueChange={(value) => setBranch(value ?? branch)} options={branches.map((item) => ({ label: item.name, value: item.name }))} placeholder="Select a branch" /> : <div><span className="mb-xs block text-label-md text-on-surface">Branch</span><p className="rounded-md border border-error/40 px-md py-sm text-body-sm text-error">No branches were found for this repository.</p></div>}
                  <Input label="Compose file" value={composePath} onChange={(event) => setComposePath(event.target.value)} placeholder="docker-compose.yml" />
                  </div>
                  {branchesError ? <p className="font-body-sm text-body-sm text-error">Could not load repository branches. Check the connected provider permissions.</p> : null}
                </>
              )}
            </div>
          )}
          {sourceMode === "editor" ? <ComposeEditor value={content} onChange={setContent} /> : null}
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
