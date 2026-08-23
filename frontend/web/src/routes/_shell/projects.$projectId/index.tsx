import { createFileRoute } from "@tanstack/react-router";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { apiPatch, apiPost } from "../../../api/client";
import type { EnvSummary } from "../../../hooks";
import {
  useProjects,
  useEnvironments,
  useCreateEnvironment,
  useUpdateEnvironment,
  useDeleteEnvironment,
  useSetDefaultEnvironment,
  useProjectApps,
  useDatabases,
  useComposeStacks,
  useAppStates,
  useEnvVars,
  useProjectVars,
  useReplaceEnvVars,
  useReplaceProjectVars,
  useDeleteApp,
  useDeleteDatabase,
  useDeleteCompose,
  useDatabaseDeploy,
  useComposeUp,
} from "../../../hooks";
import { AlertDialog, Badge, BulkActionBar, Button, Card, Dialog, EnvironmentSwitcher, Field, Input, RuntimeStatus, Skeleton, useToast } from "@aether/design-system";
import type { BadgeProps, Icon as DesignIcon } from "@aether/design-system";
import { CaretDown, CaretUp, CheckCircle, Database, Gear, Info, PencilSimple, Plus, RocketLaunch, Star, Trash, Warning } from "@phosphor-icons/react";
import { CreateServiceLauncher } from "../../../components/CreateServiceLauncher";
import { DatabaseWizard } from "../../../components/DatabaseWizard";
import { TechIcon } from "../../../components/TechIcon";
import { EnvEditorModal } from "../../../components/EnvEditorModal";
import { isRuntimeLive, mapRuntimeStatus } from "../../../lib/runtime-status";

const envSchema = z.object({
  name: z.string("Name is required").trim().min(1, "Name is required").max(40, "Maximum 40 characters"),
  description: z.string().optional(),
  color: z.string().optional(),
});

const designIcon = (icon: typeof CheckCircle) => icon as unknown as DesignIcon;

function projectStatus(statuses: string[]): { label: string; tone: NonNullable<BadgeProps["tone"]>; icon: DesignIcon } {
  const normalized = statuses.map((status) => status.trim().toLowerCase()).filter(Boolean);
  if (normalized.some((status) => ["failed", "cancelled", "rolled_back", "error", "dead"].includes(status))) {
    return { label: "Degraded", tone: "warning", icon: designIcon(Warning) };
  }
  if (normalized.some((status) => ["degraded", "warning"].includes(status))) {
    return { label: "Degraded", tone: "warning", icon: designIcon(Warning) };
  }
  if (normalized.some((status) => ["queued", "building", "starting", "health_checking", "creating", "provisioning", "syncing", "deploying"].includes(status))) {
    return { label: "Syncing", tone: "info", icon: designIcon(Info) };
  }
  if (normalized.some((status) => ["healthy", "ready", "running", "active"].includes(status))) {
    return { label: "Healthy", tone: "success", icon: designIcon(CheckCircle) };
  }
  if (normalized.some((status) => ["stopped", "exited", "no_container", "offline", "disabled", "paused"].includes(status))) {
    return { label: "Offline", tone: "neutral", icon: designIcon(Info) };
  }
  return { label: "Unknown", tone: "neutral", icon: designIcon(Info) };
}

function EnvModal({
  open,
  onClose,
  initial,
  onSubmit,
  submitting,
}: {
  open: boolean;
  onClose: () => void;
  initial?: { id: string; name: string; description: string; color: string } | null;
  onSubmit: (v: { name: string; description: string; color: string }) => void;
  submitting: boolean;
}) {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<z.input<typeof envSchema>, any, z.output<typeof envSchema>>({
    resolver: zodResolver(envSchema),
    defaultValues: { name: "", description: "", color: "" },
  });

  useEffect(() => {
    if (open) reset({ name: initial?.name ?? "", description: initial?.description ?? "", color: initial?.color ?? "" });
  }, [open, initial, reset]);

  return (
    <Dialog open={open} trigger={<span />} onOpenChange={(value) => { if (!value) onClose(); }} title={initial ? "Edit environment" : "Create environment"}>
      <form
        onSubmit={handleSubmit((v) => onSubmit({ name: v.name, description: v.description ?? "", color: v.color ?? "" }))}
        className="space-y-lg"
        noValidate
      >
        <Field label="Name" error={errors.name?.message} description="Lowercase letters, numbers and dashes.">
          <Input placeholder="staging" {...register("name")} />
        </Field>
        <Field label="Description (optional)" error={errors.description?.message}>
          <Input placeholder="pre-production validation" {...register("description")} />
        </Field>
        <Field label="Color (optional)" error={errors.color?.message} description="Hex color used in the selector, for example #ff8800.">
          <Input placeholder="#568dff" {...register("color")} />
        </Field>
        <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
          <Button type="button" variant="ghost" onClick={onClose}>Cancel</Button>
          <Button type="submit" disabled={submitting}>
            {initial ? "Save" : "Create Environment"}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}

function ProjectDetail() {
  const { projectId } = useParams({ strict: false }) as { projectId: string };
  const navigate = useNavigate();
  const search = useSearch({ strict: false }) as { environment?: string };
  const { data: projects } = useProjects();
  const { data: envs, isLoading } = useEnvironments(projectId);
  const create = useCreateEnvironment(projectId);
  const update = useUpdateEnvironment(projectId);
  const del = useDeleteEnvironment(projectId);
  const setDefault = useSetDefaultEnvironment(projectId);
  const { data: apps } = useProjectApps(projectId);
  const { data: databases } = useDatabases();
  const { data: composeStacks } = useComposeStacks();
  const { data: states } = useAppStates();
  const { add } = useToast();
  const queryClient = useQueryClient();
  const validEnvironments = useMemo(
    () => (envs ?? []).filter((environment) => Boolean(environment?.id && environment?.name?.trim())),
    [envs]
  );
  const selectedId = (() => {
    if (!validEnvironments.length) return null;
    const fromUrl = validEnvironments.find((e) => e.slug === search.environment);
    if (fromUrl) return fromUrl.id;
    const stored = localStorage.getItem(`aether.env.${projectId}`);
    const fromStorage = validEnvironments.find((e) => e.id === stored || e.slug === stored);
    if (fromStorage) return fromStorage.id;
    return validEnvironments.find((e) => e.is_default)?.id ?? validEnvironments[0].id;
  })();
  const envVars = useEnvVars(projectId, selectedId);
  const replaceEnvVars = useReplaceEnvVars(projectId, selectedId ?? "");
  const projectVars = useProjectVars(projectId);
  const replaceProjectVars = useReplaceProjectVars(projectId);
  const projectVarsList = useMemo(
    () => projectVars.data?.map((v) => ({ key: v.key, value: v.value, is_secret: v.is_secret })),
    [projectVars.data]
  );

  const project = projects?.find((p) => p.id === projectId);
  const [modal, setModal] = useState<{ open: boolean; editing: { id: string; name: string; description: string; color: string } | null }>({ open: false, editing: null });
  const [confirmDelete, setConfirmDelete] = useState<{ id: string; name: string } | null>(null);
  const [varsOpen, setVarsOpen] = useState(false);
  const [projectVarsOpen, setProjectVarsOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [dbCreateOpen, setDbCreateOpen] = useState(false);
  const [selectedServiceKeys, setSelectedServiceKeys] = useState<Set<string>>(new Set());
  const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);
  const [bulkBusy, setBulkBusy] = useState(false);
  const [bulkError, setBulkError] = useState<string | null>(null);
  const [renameTarget, setRenameTarget] = useState<{ id: string; name: string } | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const deleteApp = useDeleteApp();
  const deleteDatabase = useDeleteDatabase();
  const deleteCompose = useDeleteCompose();
  const deployApp = useMutation({ mutationFn: (id: string) => apiPost(`/api/v1/apps/${id}/deploy`) });
  const deployDatabase = useDatabaseDeploy();
  const deployCompose = useComposeUp();
  const renameApp = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => apiPatch(`/api/v1/apps/${id}`, { name }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["apps"] });
      queryClient.invalidateQueries({ queryKey: ["app", variables.id] });
      queryClient.invalidateQueries({ queryKey: ["apps", "project", projectId] });
    },
  });

  const selected = selectedId;

  const activeEnv = validEnvironments.find((e) => e.id === selected) ?? null;
  const defaultEnvId = validEnvironments.find((e) => e.is_default)?.id;
  const envApps = (apps ?? []).filter((a) =>
    a.environment_id ? a.environment_id === selected : selected === defaultEnvId
  );
  const projDatabases = (databases ?? []).filter((d) => d.project_id === projectId);
  const projCompose = (composeStacks ?? []).filter((c) => c.project_id === projectId);

  const allServices = [
    ...envApps.map((a) => {
      const state = states?.[a.id];
      return { type: "app", id: a.id, name: a.name, port: a.port, source: a.source_type, image: a.source_type === "image" ? a.image : a.git_url, runtimeStatus: mapRuntimeStatus(state, a.latest_deployment?.status), status: state ?? "" };
    }),
    ...projDatabases.map((d) => ({ type: "db", id: d.id, name: d.name, port: d.port, engine: d.engine, version: d.version, status: d.status, runtimeStatus: mapRuntimeStatus(d.status) })),
    ...projCompose.map((c) => ({ type: "compose", id: c.id, name: c.name, port: 0, source: "compose", image: "Docker Compose", status: c.status, runtimeStatus: mapRuntimeStatus(c.status) })),
  ].sort((a, b) => a.name.localeCompare(b.name));
  const serviceKeys = allServices.map((service) => `${service.type}:${service.id}`).join(",");
  const selectedServices = allServices.filter((service) => selectedServiceKeys.has(`${service.type}:${service.id}`));
  const canRenameSelected = selectedServices.length === 1 && selectedServices[0].type === "app";
  const projectStatuses = allServices.map((service) => service.runtimeStatus);

  useEffect(() => {
    const available = new Set(serviceKeys.split(",").filter(Boolean));
    setSelectedServiceKeys((current) => {
      const next = new Set([...current].filter((key) => available.has(key)));
      return next.size === current.size ? current : next;
    });
  }, [serviceKeys]);

  const toggleService = (type: string, id: string) => {
    const key = `${type}:${id}`;
    setSelectedServiceKeys((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key); else next.add(key);
      return next;
    });
    setBulkError(null);
  };

  const runBulk = async (operation: "delete" | "deploy") => {
    if (!selectedServices.length) return;
    setBulkBusy(true);
    setBulkError(null);
    const failures: string[] = [];
    await Promise.all(selectedServices.map(async (service) => {
      try {
        if (operation === "delete") {
          if (service.type === "app") await deleteApp.mutateAsync(service.id);
          else if (service.type === "db") await deleteDatabase.mutateAsync(service.id);
          else await deleteCompose.mutateAsync(service.id);
        } else if (service.type === "app") {
          await deployApp.mutateAsync(service.id);
        } else if (service.type === "db") {
          await deployDatabase.mutateAsync(service.id);
        } else {
          await deployCompose.mutateAsync(service.id);
        }
      } catch {
        failures.push(service.name);
      }
    }));
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["apps", "project", projectId] }),
      queryClient.invalidateQueries({ queryKey: ["databases"] }),
      queryClient.invalidateQueries({ queryKey: ["composes"] }),
    ]);
    setBulkBusy(false);
    setBulkDeleteOpen(false);
    if (failures.length) setBulkError(`Could not ${operation} ${failures.join(", ")}.`);
    else setSelectedServiceKeys(new Set());
  };

  const openRename = () => {
    const service = selectedServices[0];
    if (!service || service.type !== "app") return;
    setRenameTarget({ id: service.id, name: service.name });
    setRenameValue(service.name);
  };

  const submitRename = async () => {
    if (!renameTarget || !renameValue.trim()) return;
    try {
      await renameApp.mutateAsync({ id: renameTarget.id, name: renameValue.trim() });
      setRenameTarget(null);
      setSelectedServiceKeys(new Set());
      add({ title: "Service renamed", tone: "success" });
    } catch (error) {
      add({ title: "Could not rename service", description: error instanceof Error ? error.message : "Try again.", tone: "error" });
    }
  };

  const selectEnv = (id: string) => {
    const env = validEnvironments.find((e) => e.id === id);
    if (!env) return;
    localStorage.setItem(`aether.env.${projectId}`, env.slug);
    navigate({ to: "/projects/$projectId", params: { projectId }, search: { environment: env.slug } });
  };

  const submitModal = async (v: { name: string; description: string; color: string }) => {
    try {
      if (modal.editing) {
        await update.mutateAsync({ environmentID: modal.editing.id, ...v });
        add({ title: "Environment updated", tone: "success" });
      } else {
        const env = await create.mutateAsync(v);
        selectEnv(env.id);
      }
      setModal({ open: false, editing: null });
    } catch (err) {
      add({ title: "Environment update failed", description: err instanceof Error ? err.message : "Unable to save environment.", tone: "error" });
    }
  };

  const confirmDeleteEnv = async () => {
    if (!confirmDelete) return;
    try {
      await del.mutateAsync(confirmDelete.id);
      add({ title: "Environment deleted", tone: "success" });
      setConfirmDelete(null);
      const remaining = validEnvironments.filter((e) => e.id !== confirmDelete.id);
      if (remaining?.length) selectEnv(remaining.find((e) => e.is_default)?.id ?? remaining[0].id);
    } catch (err) {
      add({ title: "Environment deletion failed", description: err instanceof Error ? err.message : "Unable to delete environment.", tone: "error" });
    }
  };

  if (!project) return <Skeleton variant="card" className="min-h-48" />;

  return (
    <div className="space-y-lg">
      <div className="flex items-center justify-between gap-lg flex-wrap">
        <div className="flex items-center gap-lg min-w-0">
          <div className="min-w-0">
            <h1 className="font-headline-sm text-headline-sm text-on-surface truncate">{project.name}</h1>
            <p className="font-body-sm text-body-sm text-on-surface-variant">{validEnvironments.length} environment(s)</p>
          </div>
          <EnvironmentSwitcher
            options={validEnvironments.map((environment) => {
              const environmentApps = (apps ?? []).filter((app) => app.environment_id === environment.id).length;
              const serviceLabel = `${environmentApps} service${environmentApps === 1 ? "" : "s"}`;
              return {
              id: environment.id,
              label: environment.name,
              kind: environment.name.toLowerCase() === "production" ? "production" : environment.name.toLowerCase() === "staging" ? "staging" : "development",
              protected: environment.is_default,
              isDefault: environment.is_default,
              branch: serviceLabel,
            };
            })}
            value={selected ?? undefined}
            onValueChange={selectEnv}
            onCreate={() => setModal({ open: true, editing: null })}
            onEdit={(id) => { const environment = validEnvironments.find((item) => item.id === id); if (environment) setModal({ open: true, editing: { id: environment.id, name: environment.name, description: environment.description, color: environment.color } }); }}
            onSetDefault={(id) => setDefault.mutate(id, { onSuccess: () => add({ title: "Default environment updated", tone: "success" }) })}
            onDelete={(id) => { const environment = validEnvironments.find((item) => item.id === id); if (environment) setConfirmDelete({ id: environment.id, name: environment.name }); }}
          />
        </div>
        <ActionsMenu
            hasEnv={!!selected}
            activeEnv={activeEnv}
            onEnvVars={() => setVarsOpen(true)}
            onProjectVars={() => setProjectVarsOpen(true)}
            onCreate={() => setModal({ open: true, editing: null })}
            onRename={() => activeEnv && setModal({ open: true, editing: { id: activeEnv.id, name: activeEnv.name, description: activeEnv.description, color: activeEnv.color } })}
            onSetDefault={() => activeEnv && setDefault.mutate(activeEnv.id, { onSuccess: () => add({ title: "Default environment updated", tone: "success" }) })}
            onDelete={() => activeEnv && setConfirmDelete({ id: activeEnv.id, name: activeEnv.name })}
          />
      </div>

      {isLoading ? (
        <Skeleton variant="card" className="min-h-48" />
      ) : !activeEnv ? (
        <p className="font-body-sm text-body-sm text-on-surface-variant">No environments yet.</p>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-lg">
            <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md">
              <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-xs">Services</p>
              <p className="font-headline-sm text-headline-sm text-on-surface">{allServices.length}</p>
            </div>
            <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md">
              <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-xs">Status</p>
              {(() => {
                const status = projectStatus(projectStatuses);
                return <Badge tone={status.tone} icon={status.icon}>{status.label}</Badge>;
              })()}
            </div>
            <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md">
              <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-xs">Last deploy</p>
              <p className="font-headline-sm text-headline-sm text-on-surface">—</p>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">
              Services · {activeEnv.name}
            </h2>
            <Button onClick={() => setCreateOpen(true)}>
              <Plus size={16} />
              Create Service
            </Button>
          </div>

          <BulkActionBar
            selectedCount={selectedServices.length}
            pending={bulkBusy}
            partialFailure={bulkError}
            onClear={() => { setSelectedServiceKeys(new Set()); setBulkError(null); }}
            actions={[
              { id: "deploy", label: "Deploy", onSelect: () => void runBulk("deploy") },
              { id: "rename", label: "Rename", disabled: !canRenameSelected, onSelect: openRename },
              { id: "delete", label: "Delete", destructive: true, onSelect: () => setBulkDeleteOpen(true) },
            ]}
          />

          {allServices.length === 0 ? (
            <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md">
              <p className="font-body-sm text-body-sm text-on-surface-variant">
                No services in this environment yet. Create one in Services.
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-md">
              {allServices.map((svc) => {
                const isDb = svc.type === "db";
                const isCompose = svc.type === "compose";
                const target = isCompose ? `/compose/${svc.id}` : isDb ? `/databases/${svc.id}` : `/apps/${svc.id}`;
                const sub = isCompose ? "Docker Compose stack" : isDb ? `${(svc as { engine?: string }).engine} ${(svc as { version?: string }).version}` : (svc as { image?: string }).image ?? "";
                const serviceKey = `${svc.type}:${svc.id}`;
                return (
                  <div key={serviceKey} className="relative min-w-0">
                    <input
                      type="checkbox"
                      checked={selectedServiceKeys.has(serviceKey)}
                      onChange={() => toggleService(svc.type, svc.id)}
                      onClick={(event) => { event.stopPropagation(); }}
                      aria-label={`Select ${svc.name}`}
                      className="absolute bottom-3 left-3 z-10 size-4 accent-primary"
                    />
                    <Link
                      to={target}
                      className="group flex min-w-0 flex-col items-start gap-sm rounded-lg border border-outline-variant/50 bg-surface-container-lowest px-md pb-10 pt-5 transition-colors hover:border-primary/50 hover:bg-surface-container-high/40"
                      title={`Open ${svc.name}`}
                    >
                      <span className="flex w-full min-w-0 items-center justify-between">
                        <TechIcon name={isCompose ? "docker" : isDb ? (svc as { engine?: string }).engine : (svc as { source?: string }).source === "git" ? "gitlab" : "docker"} size={16} className="shrink-0 text-primary" />
                        <RuntimeStatus status={svc.runtimeStatus} live={isRuntimeLive(svc.runtimeStatus)} />
                      </span>
                      <span className="w-full min-w-0">
                        <span className="block truncate font-body-md text-body-md text-on-surface">{svc.name}</span>
                        <span className="block truncate font-code-md text-code-md text-on-surface-variant/60">{sub}{!isCompose && ` · :${svc.port}`}</span>
                      </span>
                      <span className="self-end text-on-surface-variant/30 transition-colors group-hover:text-primary">→</span>
                    </Link>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}

      <EnvModal
        open={modal.open}
        onClose={() => setModal({ open: false, editing: null })}
        initial={modal.editing}
        onSubmit={submitModal}
        submitting={create.isPending || update.isPending}
      />

      <CreateServiceLauncher open={createOpen} onClose={() => setCreateOpen(false)} fixedProjectId={projectId} fixedEnvironmentId={selected ?? undefined} />
      <DatabaseWizard open={dbCreateOpen} onClose={() => setDbCreateOpen(false)} fixedProjectId={projectId} />

      <Dialog open={renameTarget !== null} trigger={<span />} onOpenChange={(value) => { if (!value) setRenameTarget(null); }} title="Rename service">
        <div className="space-y-5">
          <Field label="Service name">
            <Input value={renameValue} onChange={(event) => setRenameValue(event.target.value)} autoFocus />
          </Field>
          <div className="flex justify-end gap-2 border-t border-border pt-4">
            <Button type="button" variant="ghost" onClick={() => setRenameTarget(null)}>Cancel</Button>
            <Button type="button" loading={renameApp.isPending} disabled={!renameValue.trim()} onClick={() => void submitRename()}>Rename</Button>
          </div>
        </div>
      </Dialog>

      <AlertDialog
        trigger={<span />}
        open={bulkDeleteOpen}
        onOpenChange={setBulkDeleteOpen}
        onConfirm={() => void runBulk("delete")}
        title="Delete selected services"
        description={`Delete ${selectedServices.length} selected service${selectedServices.length === 1 ? "" : "s"}? This action cannot be undone.`}
        confirmLabel="Delete"
      />

      <EnvEditorModal
        open={projectVarsOpen}
        onClose={() => setProjectVarsOpen(false)}
        title={`Project variables · ${project.name}`}
        description="Variables available to every service of this project, regardless of environment. Environment and service variables override them."
        isLoading={projectVars.isLoading}
        vars={projectVarsList}
        onSave={(entries) => replaceProjectVars.mutateAsync(entries)}
      />

      <EnvEditorModal
        open={varsOpen}
        onClose={() => setVarsOpen(false)}
        title={`Environment variables · ${activeEnv?.name ?? ""}`}
        description="Variables available to every service in this environment. Service variables override environment variables."
        isLoading={envVars.isLoading}
        vars={envVars.data}
        onSave={(entries) => replaceEnvVars.mutateAsync(entries)}
      />

      <AlertDialog
        trigger={<span />}
        open={confirmDelete !== null}
        onOpenChange={(value) => { if (!value) setConfirmDelete(null); }}
        onConfirm={confirmDeleteEnv}
        title="Delete environment"
        description={`Delete "${confirmDelete?.name}"? This is only allowed when the environment has no services.`}
        confirmLabel="Delete"
      />
    </div>
  );
}

function ActionsMenu({
  hasEnv,
  activeEnv,
  onEnvVars,
  onProjectVars,
  onCreate,
  onRename,
  onSetDefault,
  onDelete,
}: {
  hasEnv: boolean;
  activeEnv: EnvSummary | null;
  onEnvVars: () => void;
  onProjectVars: () => void;
  onCreate: () => void;
  onRename: () => void;
  onSetDefault: () => void;
  onDelete: () => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  const item = (icon: typeof Gear, label: string, onClick: () => void, danger = false, disabled = false) => {
    const Icon = icon;
    return (
    <button
      onClick={() => {
        if (disabled) return;
        setOpen(false);
        onClick();
      }}
      disabled={disabled}
      className={`w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg font-body-sm text-body-sm transition-colors ${
        danger ? "text-error hover:bg-error/10" : disabled ? "text-on-surface-variant/40 cursor-not-allowed" : "text-on-surface hover:bg-surface-container-high"
      }`}
    >
      <Icon size={16} className="w-5 shrink-0" />
      {label}
    </button>
    );
  };

  return (
    <div ref={ref} className="relative inline-block">
      <Button
        variant="secondary"
        className="py-1.5"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="font-label-caps text-label-caps uppercase">Actions</span>
        {open ? <CaretUp size={16} className="opacity-70" /> : <CaretDown size={16} className="opacity-70" />}
      </Button>
      {open && (
        <div className="absolute right-0 top-full mt-2 z-[70] w-64 rounded-xl bg-surface-popover border border-border-subtle shadow-md overflow-hidden p-1.5">
          <p className="px-2.5 pt-2 pb-1 font-label-caps text-label-caps text-on-surface-variant/60 uppercase">
            Environment · {activeEnv?.name ?? "—"}
          </p>
          {item(Gear, "Environment Variables", onEnvVars, false, !hasEnv)}
          <p className="px-2.5 pt-3 pb-1 font-label-caps text-label-caps text-on-surface-variant/60 uppercase">Project</p>
          {item(Database, "Project Variables", onProjectVars)}
          {item(Plus, "Create Environment", onCreate)}
          <div className="my-1.5 border-t border-border-subtle" />
          {item(PencilSimple, "Rename Environment", onRename, false, !hasEnv)}
          {!activeEnv?.is_default && item(Star, "Set as Default", onSetDefault, false, !hasEnv)}
          {item(Trash, "Delete Environment", onDelete, true, !hasEnv)}
        </div>
      )}
    </div>
  );
}

export const Route = createFileRoute("/_shell/projects/$projectId/")({
  component: ProjectDetail,
});
