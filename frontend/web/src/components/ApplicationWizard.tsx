import { useEffect, useRef, useState } from "react";
import { ArrowRight, CaretDown, Check, CheckCircle, Code, Database, FileArrowUp, FileText, FolderOpen, Gear, Globe, Info, LinkSimple, MagnifyingGlass, Package, Pulse, SpinnerGap, Warning, Wrench } from "@phosphor-icons/react";
import { useNavigate } from "@tanstack/react-router";
import { useCreateApp, useDisconnectGitHub, useProjects, useSourceControlBranches, useSourceControlConnections, useSourceControlRepositories, useStartGitHubManifest } from "../hooks";
import { ApiError, apiPut, getServer } from "../api/client";
import { TechIcon } from "./TechIcon";
import { AdvancedSettings } from "./AdvancedSettings";
import { Accordion, Attachment, Button, Input, Modal, NativeSelect, Select, SelectSearch, VariableEditor, type VariableRow, Wizard, useToast } from "@aether/design-system";

const wizardIcons: Record<string, typeof Code> = {
  arrow_forward: ArrowRight,
  check_circle: CheckCircle,
  code: Code,
  data_object: Database,
  description: FileText,
  expand_more: CaretDown,
  folder_open: FolderOpen,
  folder_zip: FileArrowUp,
  help: Info,
  info: Info,
  language: Globe,
  link: LinkSimple,
  radar: Pulse,
  search: MagnifyingGlass,
  settings_b_roll: Gear,
  tune: Wrench,
  warning: Warning,
  webhook: LinkSimple,
};

function WizardIcon({ name, className = "", size = 18 }: { name: string; className?: string; size?: number }) {
  const Icon = name.includes("progress_activity") ? SpinnerGap : wizardIcons[name] ?? Package;
  return <Icon size={size} className={className} aria-hidden="true" />;
}

function parseEnvVariables(): VariableRow[] | undefined {
  const pasted = window.prompt("Paste .env content (KEY=value per line):");
  if (!pasted) return;
  return pasted.split("\n").flatMap((raw, index) => {
    const line = raw.trim();
    if (!line || line.startsWith("#")) return [];
    const match = line.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
    if (!match) return [];
    return [{
      id: `imported-${index}-${match[1]}`,
      key: match[1],
      value: match[2].replace(/^"|"$/g, ""),
      secret: /password|secret|key|token/i.test(match[1]),
    }];
  });
}

export type AppKind = "web" | "api";

interface DetectedPlan {
  framework: string;
  library: string;
  package_manager: string;
  build_command: string;
  install_command: string;
  output_dir: string;
  app_type: string;
  web_server: string;
  container_port: number;
  spa_fallback: boolean;
  index_file: string;
  nginx_conf: string;
  dockerfile: string;
  warnings: string[];
  detected: boolean;
}

export function ApplicationWizard({
  open,
  onClose,
  fixedProjectId,
  fixedEnvironmentId,
  kind = "web",
}: {
  open: boolean;
  onClose: () => void;
  fixedProjectId?: string;
  fixedEnvironmentId?: string;
  kind?: AppKind;
}) {
  const navigate = useNavigate();
  const { data: projects } = useProjects();
  const { data: connections } = useSourceControlConnections();
  const startGitHubManifest = useStartGitHubManifest();
  const disconnectGitHub = useDisconnectGitHub();
  const githubConnection = connections?.find((connection) => connection.provider === "github" && connection.status === "active" && !!connection.installation_id);
  const { data: repositories, isLoading: repositoriesLoading, isError: repositoriesError, error: repositoriesQueryError, refetch: refetchRepositories } = useSourceControlRepositories(githubConnection?.installation_id);
  const githubInstallationUnavailable = repositoriesQueryError instanceof ApiError && repositoriesQueryError.status === 404;
  const createApp = useCreateApp();
  const { add } = useToast();

  const [step, setStep] = useState(1);
  const [sourceMode, setSourceMode] = useState<"git" | "upload">("git");
  const [selectedRepo, setSelectedRepo] = useState<string | null>(null);
  const [branch, setBranch] = useState("main");
  const { data: branches } = useSourceControlBranches(selectedRepo ?? undefined, githubConnection?.installation_id);
  const [customUrl, setCustomUrl] = useState("");
  const [zipUpload, setZipUpload] = useState<{ upload_id: string; name: string; size: number } | null>(null);
  const [zipFile, setZipFile] = useState<{ name: string; size: number } | null>(null);
  const [zipUploading, setZipUploading] = useState(false);

  const [projectId, setProjectId] = useState(fixedProjectId ?? "");
  const [name, setName] = useState("");
  const [buildType, setBuildType] = useState<"dockerfile" | "buildpacks" | "custom">("buildpacks");
  const [port, setPort] = useState(kind === "api" ? 8080 : 3000);
  const [installCmd, setInstallCmd] = useState("");
  const [buildCmd, setBuildCmd] = useState("");
  const [startCmd, setStartCmd] = useState(kind === "api" ? "./server" : "");
  const [dockerfilePath, setDockerfilePath] = useState("Dockerfile");
  const [rootFolder, setRootFolder] = useState("");
  const [environmentTemplatePath, setEnvironmentTemplatePath] = useState(".env.example");
  const [distFolder, setDistFolder] = useState("");
  const [watchPaths, setWatchPaths] = useState("");
  const [envRows, setEnvRows] = useState<VariableRow[]>([]);
  const [plan, setPlan] = useState<DetectedPlan | null>(null);
  const [detecting, setDetecting] = useState(false);
  const [creating, setCreating] = useState(false);

  const [cpu, setCpu] = useState("0.5");
  const [memMB, setMemMB] = useState(512);
  const [storageMB, setStorageMB] = useState(0);
  const [healthEnabled, setHealthEnabled] = useState(false);
  const [healthPath, setHealthPath] = useState("/");
  const planRef = useRef<DetectedPlan | null>(null);
  planRef.current = plan;

  useEffect(() => {
    if (!planRef.current || buildType !== "custom") return;
    const p = port;
    const t = setTimeout(async () => {
      try {
        const resp = await fetch(`${getServer()}/api/v1/plan/preview`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ plan: planRef.current, port: p }),
        });
        const data = await resp.json();
        if (resp.ok) {
          setPlan((prev) => (prev ? { ...prev, dockerfile: data.dockerfile, nginx_conf: data.nginx_conf } : prev));
        }
      } catch {
        /* preview é best-effort */
      }
    }, 400);
    return () => clearTimeout(t);
  }, [port, buildType]);

  useEffect(() => {
    if (open) {
      setStep(1);
      setSourceMode("git");
      setSelectedRepo(null);
      setBranch("main");
      setZipUpload(null);
      setZipFile(null);
      setCustomUrl("");
    }
  }, [open]);

  const selectedRepository = repositories?.find((repository) => repository.id === selectedRepo);

  const sourceReady = sourceMode === "git" ? !!selectedRepo || !!customUrl.trim() : !!zipUpload;

  const uploadZip = async (file: File) => {
    if (!file.name.toLowerCase().endsWith(".zip")) {
      add({ title: "Only .zip files are supported", tone: "error" });
      return;
    }
    setZipFile({ name: file.name, size: file.size });
    setZipUpload(null);
    setZipUploading(true);
    try {
      const fd = new FormData();
      fd.append("file", file);
      const resp = await fetch(`${getServer()}/api/v1/upload/zip`, {
        method: "POST",
        credentials: "include",
        body: fd,
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error || "upload failed");
      setZipUpload(data);
      setSelectedRepo(null);
      setCustomUrl("");
      runDetect({ upload_id: data.upload_id });
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Upload failed", tone: "error" });
    } finally {
      setZipUploading(false);
    }
  };

  const runDetect = async (source: { upload_id?: string; git_url?: string }) => {
    setDetecting(true);
    setPlan(null);
    try {
      const resp = await fetch(`${getServer()}/api/v1/analyze`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(source.upload_id ? { upload_id: source.upload_id } : { git_url: source.git_url, git_branch: "main" }),
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error || "analysis failed");
      setPlan(data);
      if (data.container_port) setPort(data.container_port);
      add({ title: `Detected: ${data.framework} (${data.app_type})`, tone: "success" });
      return true;
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Analysis failed", tone: "error" });
      return false;
    } finally {
      setDetecting(false);
    }
  };

  const detectFramework = async () => {
    const hasSource = sourceMode === "git" ? !!(customUrl.trim() || selectedRepo) : !!zipUpload;
    if (!hasSource) {
      add({ title: "Select a repository, paste a URL or upload a ZIP first", tone: "info" });
      return;
    }
    const url = customUrl.trim() || (selectedRepository ? `https://github.com/${selectedRepository.full_name}.git` : "");
    await runDetect(sourceMode === "upload" ? { upload_id: zipUpload?.upload_id } : { git_url: url });
  };

  const create = async () => {
    if (!projectId || !name.trim() || !sourceReady) {
      add({ title: "Fill in the source, project and name", tone: "error" });
      return;
    }
    setCreating(true);
    try {
      const gitUrl = customUrl.trim() || (selectedRepository ? `https://github.com/${selectedRepository.full_name}.git` : "");
      const app = await createApp.mutateAsync({
        projectID: projectId,
        payload: {
          name,
          environment_id: fixedEnvironmentId ?? "",
          source_type: sourceMode === "upload" ? "upload" : "git",
          git_url: zipUpload ? "" : gitUrl,
          git_branch: branch,
          build_type: buildType,
          dockerfile: dockerfilePath,
          upload_id: zipUpload?.upload_id ?? "",
          install_command: installCmd,
          build_command: buildCmd,
          start_command: startCmd,
          root_folder: rootFolder,
          dist_folder: distFolder,
          watch_paths: watchPaths,
          port,
          resources: { cpus: cpu, mem_mb: memMB, storage_mb: storageMB },
          health_check: { enabled: healthEnabled, path: healthPath, interval_ms: 5000, timeout_ms: 2000, retries: 3 },
          plan: plan
            ? {
                id: "",
                app_id: "",
                framework: plan.framework,
                library: plan.library,
                package_manager: plan.package_manager,
                build_command: buildType === "custom" ? (buildCmd || plan.build_command) : "",
                install_command: buildType === "custom" ? (installCmd || plan.install_command) : "",
                output_dir: plan.output_dir,
                app_type: plan.app_type,
                web_server: plan.web_server,
                container_port: port,
                spa_fallback: plan.spa_fallback,
                index_file: plan.index_file,
                nginx_conf: plan.nginx_conf,
                dockerfile: plan.dockerfile,
                warnings: plan.warnings,
              }
            : undefined,
          env: envRows
            .filter((r) => r.key.trim())
            .map((r) => ({ name: r.key.trim(), value: r.value, secret: r.secret })),
        } as never,
      });
      if (selectedRepository && githubConnection) {
        await apiPut(`/api/v1/apps/${app.id}/source`, {
          connection_id: githubConnection.id,
          repository_id: selectedRepository.id,
          repository_owner: selectedRepository.owner,
          repository_name: selectedRepository.name,
          repository_full_name: selectedRepository.full_name,
          default_branch: selectedRepository.default_branch,
          branch,
          auto_deploy: false,
          root_directory: rootFolder,
          environment_template_path: environmentTemplatePath.trim() || ".env.example",
          watch_paths: watchPaths.split(",").map((path) => path.trim()).filter(Boolean),
          ignore_paths: [],
          watch_root_files: true,
        });
      }
      add({ title: "Deploy it manually from the service page", tone: "info" });
        onClose();
        navigate({ to: "/apps/$appId", params: { appId: app.id } } as never);
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Failed to create service", tone: "error" });
    } finally {
      setCreating(false);
    }
  };

  return (
    <Modal open={open} onOpenChange={(value) => { if (!value) onClose(); }} size="wizard" showHeader={false}>
      <Wizard
        currentStep={step - 1}
        onStepChange={(value) => setStep(value + 1)}
        onCancel={onClose}
        onComplete={create}
        loading={creating}
        steps={[
          { id: "source", title: "Source", description: "Select a repository or upload source code to deploy your new service.", content: null, validate: () => sourceReady },
          { id: "configure", title: "Configure", description: "Define how the service is built and where it runs.", content: null, validate: () => Boolean(projectId && name.trim()) },
          { id: "environment", title: "Environment", description: "Set the environment variables injected at deploy time.", content: null },
        ]}
      >
        <div className="p-lg overflow-y-auto sidebar-scroll">
          {step === 1 && (
            <div className="flex flex-col gap-lg">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-md">
                {[
                  { id: "git" as const, icon: "code", label: "Git Provider", description: "Connect a provider and deploy from a repository." },
                  { id: "upload" as const, icon: "folder_zip", label: "Upload", description: "Upload a ZIP archive for this service." },
                ].map((option) => {
                  const active = sourceMode === option.id;
                  return (
                    <button
                      key={option.id}
                      type="button"
                      onClick={() => {
                        setSourceMode(option.id);
                        if (option.id === "git") setZipUpload(null);
                        else {
                          setSelectedRepo(null);
                          setCustomUrl("");
                        }
                      }}
                      className={`flex items-start gap-sm p-md rounded-xl border text-left transition-colors ${active ? "border-primary bg-primary/10" : "border-outline-variant bg-surface-container-low hover:border-primary/50"}`}
                    >
                      <WizardIcon name={option.icon} className={active ? "text-primary" : "text-on-surface-variant"} />
                      <span className="flex-1">
                        <span className="flex items-center gap-xs font-body-md text-body-md font-semibold text-on-surface">
                          {option.label}
                          {active ? <Check size={16} className="ml-auto text-primary" /> : null}
                        </span>
                        <span className="block mt-xs font-body-sm text-body-sm text-on-surface-variant">{option.description}</span>
                      </span>
                    </button>
                  );
                })}
              </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-lg">
              <div className="lg:col-span-2 flex flex-col gap-md">
                {sourceMode === "git" && <div className="bg-surface-container rounded-xl p-md border border-outline-variant hover:border-primary glow-hover transition-all duration-200">
                  <div className="flex items-start gap-md">
                    <div className="w-10 h-10 rounded-lg bg-surface flex items-center justify-center border border-outline-variant text-on-surface">
                      <Code size={18} />
                    </div>
                    <div className="flex-1 w-full">
                      <h3 className="font-body-md text-body-md font-semibold text-on-surface mb-xs">GitHub Repository</h3>
                      <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
                        Deploy directly from a connected GitHub account. Pushes will automatically trigger new builds.
                      </p>
                      {!githubConnection || githubInstallationUnavailable ? (
                        <div className="space-y-sm rounded-md border border-border bg-surface-dim p-md">
                          <p className="text-body-sm text-muted-foreground">{githubInstallationUnavailable ? "The GitHub installation is no longer available." : "Connect a GitHub App installation to search repositories."}</p>
                          <button type="button" disabled={startGitHubManifest.isPending} onClick={async () => {
                            try {
                              const manifest = await startGitHubManifest.mutateAsync({
                                return_url: `${window.location.pathname}${window.location.search}`,
                              });
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
                          }} className="text-body-sm font-semibold text-primary disabled:opacity-50">{startGitHubManifest.isPending ? "Connecting..." : "Connect GitHub"}</button>
                        </div>
                      ) : (
                        <div className="space-y-sm">
                          <SelectSearch
                            label="Repository"
                            placeholder="Search repositories"
                            value={selectedRepo}
                            options={(repositories ?? []).map((repository) => ({
                              value: repository.id,
                              label: repository.full_name,
                            }))}
                            disabled={repositoriesLoading || repositoriesError}
                            error={repositoriesError && !githubInstallationUnavailable ? (repositoriesQueryError instanceof Error ? repositoriesQueryError.message : "Could not load repositories.") : undefined}
                            onValueChange={(value) => {
                              const repository = repositories?.find((item) => item.id === value);
                              if (!repository) return;
                              setSelectedRepo(repository.id);
                              setBranch(repository.default_branch || "main");
                              setCustomUrl("");
                              setZipUpload(null);
                              if (kind === "web") runDetect({ git_url: `https://github.com/${repository.full_name}.git` });
                            }}
                          />
                          <button type="button" disabled={disconnectGitHub.isPending} onClick={() => disconnectGitHub.mutate(githubConnection.id)} className="text-body-sm font-semibold text-primary disabled:opacity-50">
                            {disconnectGitHub.isPending ? "Disconnecting..." : "Disconnect GitHub"}
                          </button>
                        </div>
                      )}
                      {githubConnection && repositoriesError && !githubInstallationUnavailable ? (
                        <button type="button" onClick={() => refetchRepositories()} className="text-body-sm font-semibold text-primary">Retry repository search</button>
                      ) : null}
                      {selectedRepository && <p className="mt-sm font-body-sm text-body-sm text-primary">Selected: {selectedRepository.full_name}</p>}
                      <div className="mt-sm">
                        <p className="font-body-sm text-body-sm text-on-surface-variant mb-xs">or paste a repository URL</p>
                        <Input placeholder="git@github.com:org/repo.git" value={customUrl} onChange={(e) => { setCustomUrl(e.target.value); setSelectedRepo(null); }} />
                      </div>
                      <div className="mt-sm flex items-center gap-sm">
                      {kind === "web" && (
                        <button
                          onClick={detectFramework}
                          disabled={detecting || (!customUrl.trim() && !selectedRepo && !zipUpload)}
                          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/10 border border-primary/30 text-primary font-label-caps text-label-caps hover:bg-primary/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          <WizardIcon name={detecting ? "progress_activity" : "radar"} className={detecting ? "animate-spin" : ""} size={16} />
                          {detecting ? "Analyzing..." : "Detect Framework"}
                        </button>
                      )}
                        {plan && (
                          <span className="font-code-md text-[11px] text-[#4ade80] flex items-center gap-1">
                            <CheckCircle size={14} />
                            {plan.framework} · {plan.app_type.toUpperCase()}
                          </span>
                        )}
                      </div>
                      {selectedRepository && (
                        <div className="mt-md space-y-md">
                          <SelectSearch
                            label="Branch"
                            value={branch}
                            placeholder="Search branches"
                            onValueChange={(value) => value && setBranch(value)}
                            options={branches?.length ? branches.map((item) => ({ label: item.name, value: item.name })) : [{ label: branch, value: branch }]}
                          />
                          <Accordion
                            items={[{
                              value: "advanced-source-settings",
                              title: "Advanced settings",
                              content: (
                                <div className="grid grid-cols-1 gap-md md:grid-cols-2">
                                  <div className="space-y-xs">
                                    <Input label="Root directory" value={rootFolder} onChange={(event) => setRootFolder(event.target.value)} placeholder="Repository root" />
                                    <p className="text-body-sm text-muted-foreground">Build only from this directory in a monorepo.</p>
                                  </div>
                                  <div className="space-y-xs">
                                    <Input label="Watch paths" value={watchPaths} onChange={(event) => setWatchPaths(event.target.value)} placeholder="apps/api/**, packages/shared/**" />
                                    <p className="text-body-sm text-muted-foreground">Only matching paths will trigger automatic builds.</p>
                                  </div>
                                  <div className="space-y-xs md:col-span-2">
                                    <Input label="Environment template file" value={environmentTemplatePath} onChange={(event) => setEnvironmentTemplatePath(event.target.value)} placeholder=".env.example" />
                                    <p className="text-body-sm text-muted-foreground">Variables found in this file will be added automatically with empty values.</p>
                                  </div>
                                </div>
                              ),
                            }]}
                          />
                        </div>
                      )}
                    </div>
                  </div>
                </div>}

                {sourceMode === "upload" && (
                  <Attachment
                    label="Build artifacts"
                    description="Upload a ZIP archive for this service."
                    accept=".zip"
                    multiple={false}
                    items={zipFile ? [{ id: "zip-upload", name: zipFile.name, size: zipFile.size, status: zipUploading ? "uploading" : zipUpload ? "complete" : "error", error: !zipUploading && !zipUpload ? "Upload failed." : undefined }] : []}
                    onFilesChange={(files) => {
                      const file = files[0];
                      if (file) uploadZip(file);
                    }}
                    onRemove={() => {
                      setZipFile(null);
                      setZipUpload(null);
                    }}
                  />
                )}
              </div>

            </div>
            </div>
          )}

          {step === 2 && (
            <div className="grid grid-cols-1 gap-md">
              <section className="bg-surface-container-low border border-outline-variant rounded-lg p-md flex flex-col gap-md relative">
                <div className="flex items-center gap-sm mb-xs">
                  <Info size={20} className="text-primary" />
                  <h2 className="font-label-caps text-label-caps text-on-surface">Application Details</h2>
                </div>
                <div className="grid grid-cols-2 gap-md">
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant">Application Name</label>
                    <input
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="my-awesome-service"
                      className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200"
                      type="text"
                    />
                  </div>
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant">Project / Environment</label>
                    <NativeSelect value={projectId} onChange={(e) => setProjectId(e.target.value)} disabled={!!fixedProjectId} options={[{ label: "Select project...", value: "" }, ...(projects ?? []).map((p) => ({ label: p.name, value: p.id }))]} />
                  </div>
                </div>
              </section>

              <section className="bg-surface-container-low border border-outline-variant rounded-lg p-md flex flex-col gap-md relative">
                <div className="flex items-center gap-sm">
                  <FileText size={20} className="text-primary" />
                  <h2 className="font-label-caps text-label-caps text-on-surface">Build Method</h2>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-sm">
                  {[
                    { id: "dockerfile", icon: "description", label: "Dockerfile", desc: "Build from a Dockerfile in the source." },
                    { id: "buildpacks", icon: "auto_awesome", label: "SmartBuild (CNB)", desc: "Auto-detect and build with CNB (Cloud Native Buildpacks)." },
                    { id: "custom", icon: "tune", label: "Custom", desc: "Manual install/build/start commands, served with nginx." },
                  ].map((bt) => {
                    const active = buildType === bt.id;
                    return (
                      <div
                        key={bt.id}
                        onClick={() => setBuildType(bt.id as typeof buildType)}
                        className={`relative ${active ? "bg-secondary-container/10 border-2 border-primary" : "bg-surface-dim border border-outline-variant hover:border-outline"} rounded p-sm flex flex-col items-start gap-xs cursor-pointer transition-colors duration-200`}
                      >
                        {active && (
                          <div className="absolute top-2 right-2">
                            <CheckCircle size={14} className="text-primary" />
                          </div>
                        )}
                        <div className="flex items-center gap-xs">
                      <WizardIcon name={bt.icon} className={active ? "text-primary" : "text-on-surface-variant"} />
                          <span className={`font-body-sm text-body-sm font-semibold ${active ? "text-on-surface" : "text-on-surface-variant"}`}>{bt.label}</span>
                        </div>
                        <p className="font-body-sm text-body-sm text-on-surface-variant/80 leading-snug">{bt.desc}</p>
                      </div>
                    );
                  })}
                </div>
                {buildType === "dockerfile" && (
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant flex items-center gap-1">
                      <FolderOpen size={14} />
                      Dockerfile path
                      <span className="font-code-md text-[10px] text-on-surface-variant/50">default: Dockerfile at repo root</span>
                    </label>
                    <Input placeholder="Dockerfile" value={dockerfilePath} onChange={(e) => setDockerfilePath(e.target.value)} />
                  </div>
                )}
                {buildType === "buildpacks" && (
                  <p className="font-body-sm text-body-sm text-on-surface-variant">
                    Auto-detects Node.js, Go, Python, Java, .NET, Ruby and more. The builder is auto-selected for the host architecture.
                  </p>
                )}
              </section>

              <section className="bg-surface-container-low border border-outline-variant rounded-lg p-md flex flex-col gap-md relative">
                <div className="flex items-center gap-sm mb-xs">
                  <Globe size={20} className="text-primary" />
                  <h2 className="font-label-caps text-label-caps text-on-surface">Public Port</h2>
                  <span className="font-code-md text-[10px] text-on-surface-variant/50">The port the service is exposed on. Auto-assigned if unavailable.</span>
                </div>
                <div className="flex items-center gap-sm">
                  <input
                    value={port}
                    onChange={(e) => setPort(parseInt(e.target.value) || 0)}
                    type="number"
                    min={1}
                    max={65535}
                    className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-32 transition-all duration-200 focus:border-primary focus:outline-none"
                  />
                  <span className="font-body-sm text-body-sm text-on-surface-variant">0 = random free port</span>
                </div>
              </section>

              <section className={`bg-surface-container-low border border-outline-variant rounded-lg p-md flex flex-col gap-md relative ${buildType === "custom" ? "" : "opacity-50 pointer-events-none"}`}>
                <div className="flex items-center justify-between mb-xs">
                  <div className="flex items-center gap-sm">
                    <Wrench size={20} className="text-secondary" />
                    <h2 className="font-label-caps text-label-caps text-on-surface">Build Configuration</h2>
                  </div>
                </div>
                <div className="grid grid-cols-1 gap-md">
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant flex items-center gap-1">
                      Install Command
                      <Info size={14} className="text-outline cursor-help" />
                    </label>
                    <input value={installCmd} onChange={(e) => setInstallCmd(e.target.value)} placeholder="npm ci" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                  </div>
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant flex items-center gap-1">
                      Build Command
                      <Info size={14} className="text-outline cursor-help" />
                    </label>
                    <input value={buildCmd} onChange={(e) => setBuildCmd(e.target.value)} placeholder="npm run build" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                  </div>
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant flex items-center gap-1">
                      Start Command
                      <Info size={14} className="text-outline cursor-help" />
                    </label>
                    <input value={startCmd} onChange={(e) => setStartCmd(e.target.value)} placeholder="npm start" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                  </div>
                </div>
                <div className="mt-sm border-t border-outline-variant/50 pt-sm">
                  <Accordion
                    items={[{
                      value: "advanced-build-settings",
                      title: "Advanced build settings",
                      content: <Input label="Dist folder" value={distFolder} onChange={(event) => setDistFolder(event.target.value)} placeholder="dist" />,
                    }]}
                  />
                </div>
              </section>

              {plan && (
                <section className="bg-surface-container-low border border-primary/30 rounded-lg p-md flex flex-col gap-md relative">
                  <div className="flex items-center gap-sm mb-xs">
                  <Pulse size={20} className="text-primary" />
                    <h2 className="font-label-caps text-label-caps text-on-surface">Detected Stack</h2>
                    <span className="ml-auto font-code-md text-[10px] text-on-surface-variant/70">auto-detected · editable</span>
                  </div>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3 font-body-sm text-body-sm">
                    {[
                      ["Framework", plan.framework], ["Library", plan.library],
                      ["Package Manager", plan.package_manager],
                      ["Build Command", plan.build_command], ["Output", plan.output_dir],
                      ["Application Type", plan.app_type], ["Web Server", plan.web_server],
                      ["Container Port", String(port)], ["SPA Fallback", plan.spa_fallback ? "Enabled" : "Disabled"],
                    ].map(([k, v]) => (
                      <div key={k} className="flex flex-col">
                        <span className="font-label-caps text-[9px] text-on-surface-variant/60 uppercase tracking-wider">{k}</span>
                        <span className="font-code-md text-[11px] text-on-surface truncate" title={v}>{v || "—"}</span>
                      </div>
                    ))}
                  </div>
                  {buildType === "custom" && (
                  <div className="flex flex-col gap-3 mt-sm pt-sm border-t border-outline-variant/40">
                    <details className="group">
                      <summary className="flex items-center gap-1 font-code-md text-[11px] text-primary cursor-pointer select-none">
                        <CaretDown size={14} className="transition-transform group-open:rotate-180" />
                        Preview generated nginx.conf
                      </summary>
                      <pre className="mt-sm bg-[#050505] border border-white/10 rounded-lg p-3 font-code-md text-[11px] text-on-surface overflow-auto max-h-56 whitespace-pre-wrap">{plan.nginx_conf}</pre>
                    </details>
                    <details className="group">
                      <summary className="flex items-center gap-1 font-code-md text-[11px] text-primary cursor-pointer select-none">
                        <CaretDown size={14} className="transition-transform group-open:rotate-180" />
                        Preview generated Dockerfile
                      </summary>
                      <pre className="mt-sm bg-[#050505] border border-white/10 rounded-lg p-3 font-code-md text-[11px] text-on-surface overflow-auto max-h-56 whitespace-pre-wrap">{plan.dockerfile}</pre>
                    </details>
                    {plan.warnings?.length > 0 && (
                      <div className="flex items-start gap-1 text-[#fbbf24]">
                  <Warning size={14} />
                        <span className="font-code-md text-[11px]">{plan.warnings[0]}</span>
                      </div>
                    )}
                  </div>
                  )}
                </section>
              )}
            </div>
          )}

          {step === 3 && (
            <div className="flex flex-col gap-lg">
              <VariableEditor
                variables={envRows}
                onChange={setEnvRows}
                onImport={parseEnvVariables}
                onExport={() => {
                  const content = envRows
                    .filter((variable) => variable.key.trim())
                    .map((variable) => `${variable.key}=${variable.value}`)
                    .join("\n");
                  void navigator.clipboard?.writeText(content);
                }}
              />
              <AdvancedSettings
                values={{ cpu, memMB, storageMB, healthEnabled, healthPath }}
                onChange={(v) => {
                  setCpu(v.cpu);
                  setMemMB(v.memMB);
                  setStorageMB(v.storageMB);
                  setHealthEnabled(v.healthEnabled);
                  setHealthPath(v.healthPath);
                }}
              />
            </div>
          )}
        </div>

      </Wizard>
    </Modal>
  );
}
