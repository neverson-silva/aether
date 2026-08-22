import { useEffect, useRef, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useCreateApp, useProjects, useSourceControlBranches, useSourceControlConnections, useSourceControlRepositories, useStartGitHubManifest } from "../hooks";
import { apiPut, getServer } from "../api/client";
import { useOverlayGate } from "./OverlayManager";
import { TechIcon } from "./TechIcon";
import { AdvancedSettings } from "./AdvancedSettings";
import { EnvRowsEditor, type EnvRowInput } from "./EnvRowsEditor";
import { useScopeVariables } from "./useScopeVariables";
import { Button, Input, Select, useToast } from "./ui";

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
  const githubConnection = connections?.find((connection) => connection.provider === "github" && connection.status === "active" && !!connection.installation_id);
  const { data: repositories, isLoading: repositoriesLoading, isError: repositoriesError, error: repositoriesQueryError, refetch: refetchRepositories } = useSourceControlRepositories(githubConnection?.installation_id);
  const createApp = useCreateApp();
  const { toast } = useToast();

  const [step, setStep] = useState(1);
  const [sourceMode, setSourceMode] = useState<"git" | "upload">("git");
  const [repoQuery, setRepoQuery] = useState("");
  const [repoPickerOpen, setRepoPickerOpen] = useState(false);
  const repoPickerRef = useRef<HTMLDivElement>(null);
  const [selectedRepo, setSelectedRepo] = useState<string | null>(null);
  const [branch, setBranch] = useState("main");
  const { data: branches, isLoading: branchesLoading } = useSourceControlBranches(selectedRepo ?? undefined, githubConnection?.installation_id);
  const [customUrl, setCustomUrl] = useState("");
  const [zipUpload, setZipUpload] = useState<{ upload_id: string; name: string; size: number } | null>(null);
  const [zipUploading, setZipUploading] = useState(false);
  const zipInputRef = useRef<HTMLInputElement>(null);

  const [projectId, setProjectId] = useState(fixedProjectId ?? "");
  const [name, setName] = useState("");
  const [buildType, setBuildType] = useState<"dockerfile" | "buildpacks" | "custom">("buildpacks");
  const [port, setPort] = useState(kind === "api" ? 8080 : 3000);
  const [installCmd, setInstallCmd] = useState("");
  const [buildCmd, setBuildCmd] = useState("");
  const [startCmd, setStartCmd] = useState(kind === "api" ? "./server" : "");
  const [dockerfilePath, setDockerfilePath] = useState("Dockerfile");
  const [rootFolder, setRootFolder] = useState("");
  const [distFolder, setDistFolder] = useState("");
  const [watchPaths, setWatchPaths] = useState("");
  const [envRows, setEnvRows] = useState<EnvRowInput[]>([]);
  const scopeGroups = useScopeVariables();
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

  const { mounted, closing, close } = useOverlayGate("app-wizard", open, onClose);

  useEffect(() => {
    if (open) {
      setStep(1);
      setSourceMode("git");
      setSelectedRepo(null);
      setBranch("main");
      setZipUpload(null);
      setRepoQuery("");
      setCustomUrl("");
    }
  }, [open]);

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (!repoPickerRef.current?.contains(event.target as Node)) setRepoPickerOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  if (!mounted) return null;

  const filteredRepos = (repositories ?? []).filter(
    (r) => !repoQuery || r.full_name.toLowerCase().includes(repoQuery.toLowerCase()) || r.name.toLowerCase().includes(repoQuery.toLowerCase())
  );
  const selectedRepository = repositories?.find((repository) => repository.id === selectedRepo);

  const sourceReady = sourceMode === "git" ? !!selectedRepo || !!customUrl.trim() : !!zipUpload;

  const uploadZip = async (file: File) => {
    if (!file.name.toLowerCase().endsWith(".zip")) {
      toast("Only .zip files are supported", "error");
      return;
    }
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
      toast(err instanceof Error ? err.message : "upload failed", "error");
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
      toast(`Detected: ${data.framework} (${data.app_type})`);
      return true;
    } catch (err) {
      toast(err instanceof Error ? err.message : "analysis failed", "error");
      return false;
    } finally {
      setDetecting(false);
    }
  };

  const detectFramework = async () => {
    const hasSource = sourceMode === "git" ? !!(customUrl.trim() || selectedRepo) : !!zipUpload;
    if (!hasSource) {
      toast("Select a repository, paste a URL or upload a ZIP first", "info");
      return;
    }
    const url = customUrl.trim() || (selectedRepository ? `https://github.com/${selectedRepository.full_name}.git` : "");
    await runDetect(sourceMode === "upload" ? { upload_id: zipUpload?.upload_id } : { git_url: url });
  };

  const create = async () => {
    if (!projectId || !name.trim() || !sourceReady) {
      toast("Fill in the source, project and name", "error");
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
          auto_deploy: true,
          root_directory: rootFolder,
          watch_paths: watchPaths.split(",").map((path) => path.trim()).filter(Boolean),
          ignore_paths: [],
          watch_root_files: true,
        });
      }
        toast("Deploy it manually from the service page", "info");
        onClose();
        navigate({ to: "/apps/$appId", params: { appId: app.id } } as never);
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create service", "error");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className={`fixed inset-0 z-[80] flex items-center justify-center modal-overlay p-4 ${closing ? "animate-fade-out" : "animate-fade-in"}`} onClick={() => close()}>
      <div className="fixed inset-0 pointer-events-none">
        <div className="absolute top-[10%] left-[20%] w-[40%] h-[40%] rounded-full bg-primary/10 blur-[150px]" />
        <div className="absolute bottom-[10%] right-[20%] w-[30%] h-[30%] rounded-full bg-secondary/10 blur-[120px]" />
      </div>
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Application wizard"
        onClick={(e) => e.stopPropagation()}
        className="glass-panel rounded-xl w-full max-w-[1000px] max-h-[90vh] flex flex-col overflow-hidden relative animate-modal-pop"
      >
        <button onClick={() => close()} className="absolute top-md right-md text-outline-variant hover:text-on-surface transition-colors z-10">
          <span className="material-symbols-outlined">close</span>
        </button>

        <div className="p-lg border-b border-outline-variant bg-surface-container-low/50">
          <div className="flex items-center gap-sm mb-xs text-on-surface-variant font-label-caps text-label-caps">
            <span>Create Service</span>
            <span className="material-symbols-outlined" style={{ fontSize: 14 }}>chevron_right</span>
            <span className={`px-sm py-[2px] rounded-full ${step === 1 ? "text-primary bg-primary/10" : "opacity-50"}`}>1. Source</span>
            <span className="material-symbols-outlined" style={{ fontSize: 14 }}>arrow_right_alt</span>
            <span className={`px-sm py-[2px] rounded-full ${step === 2 ? "text-primary bg-primary/10 border border-primary/30" : "opacity-50"}`}>2. Configure</span>
            <span className="material-symbols-outlined" style={{ fontSize: 14 }}>arrow_right_alt</span>
            <span className={`px-sm py-[2px] rounded-full ${step === 3 ? "text-primary bg-primary/10" : "opacity-50"}`}>3. Environment</span>
          </div>
          <h2 className="font-headline-sm text-lg md:text-headline-sm font-semibold text-on-surface flex items-center gap-sm">
            <span className="material-symbols-outlined text-primary">{step === 1 ? "webhook" : step === 2 ? "settings_b_roll" : "data_object"}</span>
            {step === 1 ? "Connect Source" : step === 2 ? "Configure" : "Environment Variables"}
          </h2>
          <p className="font-body-md text-body-md text-on-surface-variant mt-xs">
            {step === 1
              ? "Select a repository or upload source code to deploy your new service."
              : step === 2
                ? "Define how the service is built and where it runs."
                : "Set the environment variables injected at deploy time."}
          </p>
        </div>

        <div className="p-lg overflow-y-auto flex-1 sidebar-scroll">
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
                      <span className={`material-symbols-outlined ${active ? "text-primary" : "text-on-surface-variant"}`}>{option.icon}</span>
                      <span className="flex-1">
                        <span className="flex items-center gap-xs font-body-md text-body-md font-semibold text-on-surface">
                          {option.label}
                          <span className={`material-symbols-outlined ml-auto text-[16px] ${active ? "text-primary" : "text-outline-variant"}`}>{active ? "check_box" : "check_box_outline_blank"}</span>
                        </span>
                        <span className="block mt-xs font-body-sm text-body-sm text-on-surface-variant">{option.description}</span>
                      </span>
                    </button>
                  );
                })}
              </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-lg">
              <div className="lg:col-span-2 flex flex-col gap-md">
                {sourceMode === "git" && <div className="bg-surface-container rounded-xl p-md border border-outline-variant hover:border-primary glow-hover transition-all duration-200">
                  <div className="flex items-start gap-md">
                    <div className="w-10 h-10 rounded-lg bg-surface flex items-center justify-center border border-outline-variant text-on-surface">
                      <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 1" }}>code</span>
                    </div>
                    <div className="flex-1">
                      <h3 className="font-body-md text-body-md font-semibold text-on-surface mb-xs">GitHub Repository</h3>
                      <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
                        Deploy directly from a connected GitHub account. Pushes will automatically trigger new builds.
                      </p>
                      <div ref={repoPickerRef} onMouseDown={() => setRepoPickerOpen(true)} className="group relative bg-surface-dim rounded-lg border border-outline-variant overflow-visible">
                        <div className="flex items-center gap-sm p-sm border-b border-outline-variant bg-surface-container-low">
                          <span className="material-symbols-outlined text-outline" style={{ fontSize: 18 }}>search</span>
                          <input
                            value={repoQuery}
                            onChange={(e) => setRepoQuery(e.target.value)}
                            onFocus={() => setRepoPickerOpen(true)}
                            onClick={() => setRepoPickerOpen(true)}
                            placeholder="Search aether-labs repos..."
                            className="bg-transparent border-none outline-none focus:ring-0 text-on-surface font-body-sm text-body-sm w-full placeholder:text-outline-variant py-0"
                          />
                        </div>
                        <div className="relative mt-xs hidden max-h-64 overflow-y-auto repo-scroll rounded-lg border border-outline-variant bg-surface-container-high p-xs shadow-xl group-focus-within:block">
                          {!githubConnection && (
                            <div className="p-sm space-y-sm">
                              <p className="font-body-sm text-body-sm text-on-surface-variant">Connect a GitHub App installation to list repositories.</p>
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
                                  toast(error instanceof Error ? error.message : "GitHub connection failed", "error");
                                }
                              }} className="text-primary font-body-sm text-body-sm disabled:opacity-50">{startGitHubManifest.isPending ? "Connecting..." : "Connect GitHub"}</button>
                            </div>
                          )}
                          {githubConnection && repositoriesLoading && (
                            <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">Loading repositories...</p>
                          )}
                          {githubConnection && repositoriesError && (
                            <div className="p-sm flex items-center justify-between gap-sm">
                              <p className="font-body-sm text-body-sm text-error">{repositoriesQueryError instanceof Error ? repositoriesQueryError.message : "Could not load repositories."}</p>
                              <button type="button" onClick={() => refetchRepositories()} className="text-primary font-body-sm text-body-sm">Retry</button>
                            </div>
                          )}
                          {githubConnection && !repositoriesLoading && !repositoriesError && filteredRepos.length === 0 && (
                            <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">No repositories match "{repoQuery}".</p>
                          )}
                          {filteredRepos.map((r) => (
                            <div
                              key={r.id}
                              onMouseDown={() => {
                                setSelectedRepo(r.id);
                                setBranch(r.default_branch || "main");
                                setRepoQuery(r.full_name);
                                setCustomUrl("");
                                setZipUpload(null);
                                setRepoPickerOpen(false);
                                if (kind === "web") {
                                  runDetect({ git_url: `https://github.com/${r.full_name}.git` });
                                }
                              }}
                              className={`p-sm rounded-lg hover:bg-surface-variant flex items-center justify-between cursor-pointer border transition-colors group ${
                                selectedRepo === r.id ? "border-primary bg-primary/5" : "border-transparent hover:border-outline-variant"
                              }`}
                            >
                              <div className="flex flex-col gap-xs">
                                <div className="flex items-center gap-xs font-code-md text-code-md text-on-surface group-hover:text-primary transition-colors">
                                  <span className="material-symbols-outlined text-outline" style={{ fontSize: 16 }}>code</span>
                                  {r.full_name}
                                  {selectedRepo === r.id && <span className="material-symbols-outlined text-primary" style={{ fontSize: 14 }}>check_circle</span>}
                                </div>
                                <div className="flex items-center gap-md font-body-sm text-body-sm text-outline-variant">
                                  <span className="flex items-center gap-xs"><span className="w-2 h-2 rounded-full bg-primary" /> GitHub</span>
                                  <span>Default: {r.default_branch || "main"}</span>
                                </div>
                              </div>
                              <button type="button" className={`px-sm py-xs bg-surface border border-outline-variant rounded text-on-surface font-label-caps text-label-caps hover:bg-surface-variant transition-colors ${selectedRepo === r.id ? "text-primary border-primary/50" : "opacity-0 group-hover:opacity-100"}`}>
                                {selectedRepo === r.id ? "Selected" : "Select"}
                              </button>
                            </div>
                          ))}
                        </div>
                      </div>
                      {selectedRepository && <p className="mt-sm font-body-sm text-body-sm text-primary">Selected: {selectedRepository.full_name}</p>}
                      <div className="mt-sm">
                        <p className="font-body-sm text-body-sm text-on-surface-variant mb-xs">or paste a repository URL</p>
                        <Input icon="link" placeholder="git@github.com:org/repo.git" value={customUrl} onChange={(e) => { setCustomUrl(e.target.value); setSelectedRepo(null); }} />
                      </div>
                      <div className="mt-sm flex items-center gap-sm">
                      {kind === "web" && (
                        <button
                          onClick={detectFramework}
                          disabled={detecting || (!customUrl.trim() && !selectedRepo && !zipUpload)}
                          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/10 border border-primary/30 text-primary font-label-caps text-label-caps hover:bg-primary/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          <span className="material-symbols-outlined text-[16px]">{detecting ? "progress_activity animate-spin" : "radar"}</span>
                          {detecting ? "Analyzing..." : "Detect Framework"}
                        </button>
                      )}
                        {plan && (
                          <span className="font-code-md text-[11px] text-[#4ade80] flex items-center gap-1">
                            <span className="material-symbols-outlined text-[14px]">check_circle</span>
                            {plan.framework} · {plan.app_type.toUpperCase()}
                          </span>
                        )}
                      </div>
                      {selectedRepository && (
                        <div className="mt-md grid grid-cols-1 md:grid-cols-2 gap-md">
                          <div className="flex flex-col gap-xs">
                            <label className="font-label-caps text-label-caps text-on-surface-variant">Branch</label>
                            <Select value={branch} onChange={(event) => setBranch(event.target.value)} disabled={branchesLoading}>
                              {branches?.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}
                              {!branches?.length && <option value={branch}>{branchesLoading ? "Loading branches..." : branch}</option>}
                            </Select>
                          </div>
                          <div className="flex flex-col gap-xs">
                            <label className="font-label-caps text-label-caps text-on-surface-variant">Root directory</label>
                            <Input value={rootFolder} onChange={(event) => setRootFolder(event.target.value)} placeholder="Repository root" />
                            <p className="font-body-sm text-body-sm text-on-surface-variant">Build only from this directory in a monorepo.</p>
                          </div>
                        </div>
                      )}
                      {selectedRepository && (
                        <details className="mt-md group">
                          <summary className="flex items-center gap-xs text-primary font-body-sm text-body-sm cursor-pointer select-none">Change default root and watch paths</summary>
                          <div className="mt-sm flex flex-col gap-sm">
                            <Input value={watchPaths} onChange={(event) => setWatchPaths(event.target.value)} placeholder="apps/api/**, packages/shared/**" />
                            <p className="font-body-sm text-body-sm text-on-surface-variant">Only matching paths will trigger automatic builds.</p>
                          </div>
                        </details>
                      )}
                    </div>
                  </div>
                </div>}

                {sourceMode === "upload" && <div
                  onClick={() => zipInputRef.current?.click()}
                  className="bg-surface-container rounded-xl p-md border border-outline-variant hover:border-primary glow-hover cursor-pointer transition-all duration-200"
                >
                  <div className="flex items-center gap-md">
                    <div className="w-10 h-10 rounded-lg bg-surface flex items-center justify-center border border-outline-variant text-on-surface-variant">
                      <span className="material-symbols-outlined">folder_zip</span>
                    </div>
                    <div className="flex-1">
                      <h3 className="font-body-md text-body-md font-semibold text-on-surface mb-xs">Upload ZIP</h3>
                      <p className="font-body-sm text-body-sm text-on-surface-variant">
                        {zipUpload ? `"${zipUpload.name}" (${(zipUpload.size / 1024).toFixed(0)} KB) — ready to deploy` : "Deploy an isolated package manually. (.zip up to 64MB)"}
                      </p>
                    </div>
                    {zipUploading ? (
                      <span className="material-symbols-outlined animate-spin text-primary">progress_activity</span>
                    ) : zipUpload ? (
                      <span className="material-symbols-outlined text-[#4ade80]">check_circle</span>
                    ) : (
                      <span className="material-symbols-outlined text-outline-variant">chevron_right</span>
                    )}
                  </div>
                  <input
                    ref={zipInputRef}
                    type="file"
                    accept=".zip"
                    className="hidden"
                    onChange={(e) => {
                      const f = e.target.files?.[0];
                      if (f) uploadZip(f);
                      e.target.value = "";
                    }}
                  />
                </div>}
              </div>

            </div>
            </div>
          )}

          {step === 2 && (
            <div className="grid grid-cols-1 gap-md">
              <section className="bg-surface-container-low border border-outline-variant rounded-lg p-md flex flex-col gap-md relative">
                <div className="flex items-center gap-sm mb-xs">
                  <span className="material-symbols-outlined text-primary text-xl">info</span>
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
                    <Select
                      value={projectId}
                      onChange={(e) => setProjectId(e.target.value)}
                      disabled={!!fixedProjectId}
                      className="bg-surface-dim text-on-surface font-body-md text-body-md"
                    >
                      <option value="">Select project...</option>
                      {(projects ?? []).map((p) => (
                        <option key={p.id} value={p.id}>{p.name}</option>
                      ))}
                    </Select>
                  </div>
                </div>
              </section>

              <section className="bg-surface-container-low border border-outline-variant rounded-lg p-md flex flex-col gap-md relative">
                <div className="flex items-center gap-sm">
                  <span className="material-symbols-outlined text-primary text-xl">description</span>
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
                            <span className="material-symbols-outlined text-primary text-sm" style={{ fontVariationSettings: "'FILL' 1" }}>check_circle</span>
                          </div>
                        )}
                        <div className="flex items-center gap-xs">
                          <span className={`material-symbols-outlined ${active ? "text-primary" : "text-on-surface-variant"}`} style={{ fontSize: 18 }}>{bt.icon}</span>
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
                      <span className="material-symbols-outlined text-[14px]">folder_open</span>
                      Dockerfile path
                      <span className="font-code-md text-[10px] text-on-surface-variant/50">default: Dockerfile at repo root</span>
                    </label>
                    <Input icon="folder_open" placeholder="Dockerfile" value={dockerfilePath} onChange={(e) => setDockerfilePath(e.target.value)} />
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
                  <span className="material-symbols-outlined text-primary text-xl">language</span>
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
                    <span className="material-symbols-outlined text-secondary text-xl">build</span>
                    <h2 className="font-label-caps text-label-caps text-on-surface">Build Configuration</h2>
                  </div>
                </div>
                <div className="grid grid-cols-1 gap-md">
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant flex items-center gap-1">
                      Install Command
                      <span className="material-symbols-outlined text-[14px] text-outline cursor-help" title="Command to install dependencies.">help</span>
                    </label>
                    <input value={installCmd} onChange={(e) => setInstallCmd(e.target.value)} placeholder="npm ci" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                  </div>
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant flex items-center gap-1">
                      Build Command
                      <span className="material-symbols-outlined text-[14px] text-outline cursor-help" title="Command to compile your application.">help</span>
                    </label>
                    <input value={buildCmd} onChange={(e) => setBuildCmd(e.target.value)} placeholder="npm run build" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                  </div>
                  <div className="flex flex-col gap-xs">
                    <label className="font-label-caps text-label-caps text-on-surface-variant flex items-center gap-1">
                      Start Command
                      <span className="material-symbols-outlined text-[14px] text-outline cursor-help" title="Command to start the production server.">help</span>
                    </label>
                    <input value={startCmd} onChange={(e) => setStartCmd(e.target.value)} placeholder="npm start" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                  </div>
                </div>
                <div className="mt-sm pt-sm border-t border-outline-variant/50">
                  <button
                    type="button"
                    onClick={() => document.getElementById("advanced-settings")?.classList.toggle("hidden")}
                    className="flex items-center gap-xs text-primary hover:text-primary-fixed-dim transition-colors duration-150 font-body-sm text-body-sm"
                  >
                    <span className="material-symbols-outlined text-sm">tune</span>
                    Advanced Directory Settings
                  </button>
                  <div className="hidden grid grid-cols-2 gap-md mt-md" id="advanced-settings">
                    <div className="flex flex-col gap-xs">
                      <label className="font-label-caps text-label-caps text-on-surface-variant">Root Folder</label>
                      <input value={rootFolder} onChange={(e) => setRootFolder(e.target.value)} placeholder="./" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                    </div>
                    <div className="flex flex-col gap-xs">
                      <label className="font-label-caps text-label-caps text-on-surface-variant">Dist Folder</label>
                      <input value={distFolder} onChange={(e) => setDistFolder(e.target.value)} placeholder="dist" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                    </div>
                    <div className="flex flex-col gap-xs col-span-2">
                      <label className="font-label-caps text-label-caps text-on-surface-variant">Watch Paths (Monorepo)</label>
                      <input value={watchPaths} onChange={(e) => setWatchPaths(e.target.value)} placeholder="apps/api/**, packages/shared/**" className="bg-surface-dim border border-outline-variant text-on-surface font-code-md text-code-md px-sm py-xs rounded w-full transition-all duration-200" type="text" />
                    </div>
                  </div>
                </div>
              </section>

              {plan && (
                <section className="bg-surface-container-low border border-primary/30 rounded-lg p-md flex flex-col gap-md relative">
                  <div className="flex items-center gap-sm mb-xs">
                    <span className="material-symbols-outlined text-primary text-xl" style={{ fontVariationSettings: "'FILL' 1" }}>radar</span>
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
                        <span className="material-symbols-outlined text-[14px] transition-transform group-open:rotate-180">expand_more</span>
                        Preview generated nginx.conf
                      </summary>
                      <pre className="mt-sm bg-[#050505] border border-white/10 rounded-lg p-3 font-code-md text-[11px] text-on-surface overflow-auto max-h-56 whitespace-pre-wrap">{plan.nginx_conf}</pre>
                    </details>
                    <details className="group">
                      <summary className="flex items-center gap-1 font-code-md text-[11px] text-primary cursor-pointer select-none">
                        <span className="material-symbols-outlined text-[14px] transition-transform group-open:rotate-180">expand_more</span>
                        Preview generated Dockerfile
                      </summary>
                      <pre className="mt-sm bg-[#050505] border border-white/10 rounded-lg p-3 font-code-md text-[11px] text-on-surface overflow-auto max-h-56 whitespace-pre-wrap">{plan.dockerfile}</pre>
                    </details>
                    {plan.warnings?.length > 0 && (
                      <div className="flex items-start gap-1 text-[#fbbf24]">
                        <span className="material-symbols-outlined text-[14px]">warning</span>
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
            <div className="flex flex-col gap-md max-w-[42rem] mx-auto">
              <div className="bg-surface-container rounded-xl p-md border border-outline-variant">
                <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Environment variables</p>
                <EnvRowsEditor value={envRows} onChange={setEnvRows} compact groups={scopeGroups} />
              </div>
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

        <div className="p-lg border-t border-outline-variant bg-surface-container-low/50 flex justify-between items-center">
          <Button
            variant="ghost"
            onClick={() => (step > 1 ? setStep(step - 1) : close())}
          >
            {step > 1 ? "Back to " + (step === 2 ? "Source" : "Configure") : "Cancel"}
          </Button>
          <div className="flex items-center gap-sm">
            {step === 2 ? (
              <>
                <Button onClick={() => setStep(step + 1)} disabled={!projectId || !name.trim()}>
                  Next: Environment Variables
                  <span className="material-symbols-outlined" style={{ fontSize: 18 }}>arrow_forward</span>
                </Button>
              </>
            ) : step === 1 ? (
              <Button onClick={() => setStep(step + 1)} disabled={!sourceReady}>
                Continue
                <span className="material-symbols-outlined" style={{ fontSize: 18 }}>arrow_forward</span>
              </Button>
            ) : (
              <Button onClick={() => create()} disabled={creating}>
                {creating ? "Creating..." : "Create Service"}
                <span className="material-symbols-outlined" style={{ fontSize: 18 }}>arrow_forward</span>
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
