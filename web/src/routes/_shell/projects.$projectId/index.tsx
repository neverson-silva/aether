import { createFileRoute } from "@tanstack/react-router";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
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
} from "../../../hooks";
import { Button, ConfirmDialog, Field, Input, Modal, Spinner, StatusPill, useToast } from "../../../components/ui";
import { CreateServiceLauncher } from "../../../components/CreateServiceLauncher";
import { DatabaseWizard } from "../../../components/DatabaseWizard";
import { TechIcon } from "../../../components/TechIcon";
import { EnvEditorModal } from "../../../components/EnvEditorModal";
import { EnvironmentSwitcher } from "./-components/EnvironmentSwitcher";

const envSchema = z.object({
  name: z.string("Name is required").trim().min(1, "Name is required").max(40, "Maximum 40 characters"),
  description: z.string().optional(),
  color: z.string().optional(),
});

function tone(status: string): string {
  return status === "healthy" ? "active" : status === "degraded" ? "failed" : status === "syncing" ? "pending" : "disabled";
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
    <Modal open={open} onClose={onClose} title={initial ? "Edit environment" : "Create environment"}>
      <form
        onSubmit={handleSubmit((v) => onSubmit({ name: v.name, description: v.description ?? "", color: v.color ?? "" }))}
        className="space-y-lg"
        noValidate
      >
        <Field label="Name" hint={errors.name?.message || "lowercase letters, numbers and dashes"}>
          <Input icon="deployed_code" placeholder="staging" {...register("name")} />
        </Field>
        <Field label="Description (optional)" hint={errors.description?.message}>
          <Input icon="notes" placeholder="pre-production validation" {...register("description")} />
        </Field>
        <Field label="Color (optional)" hint={errors.color?.message || "hex color used in the selector, e.g. #ff8800"}>
          <Input icon="palette" placeholder="#568dff" {...register("color")} />
        </Field>
        <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
          <Button type="button" variant="ghost" onClick={onClose}>Cancel</Button>
          <Button type="submit" disabled={submitting}>
            {initial ? "Save" : "Create Environment"}
          </Button>
        </div>
      </form>
    </Modal>
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
  const { toast } = useToast();
  const selectedId = (() => {
    if (!envs?.length) return null;
    const fromUrl = envs.find((e) => e.slug === search.environment);
    if (fromUrl) return fromUrl.id;
    const stored = localStorage.getItem(`aether.env.${projectId}`);
    const fromStorage = envs.find((e) => e.id === stored || e.slug === stored);
    if (fromStorage) return fromStorage.id;
    return envs.find((e) => e.is_default)?.id ?? envs[0].id;
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

  const selected = selectedId;

  const activeEnv = envs?.find((e) => e.id === selected) ?? null;
  const defaultEnvId = envs?.find((e) => e.is_default)?.id;
  const envApps = (apps ?? []).filter((a) =>
    a.environment_id ? a.environment_id === selected : selected === defaultEnvId
  );
  const projDatabases = (databases ?? []).filter((d) => d.project_id === projectId);
  const projCompose = (composeStacks ?? []).filter((c) => c.project_id === projectId);

  const allServices = [
    ...envApps.map((a) => {
      const state = states?.[a.id];
      const pill =
        state === "running"
          ? { status: "running", pulse: true }
          : state === "paused"
            ? { status: "paused", pulse: false }
            : state === "dead" || state === "error"
              ? { status: "error", pulse: false }
              : { status: "provisioning", pulse: true };
      return { type: "app", id: a.id, name: a.name, port: a.port, source: a.source_type, image: a.source_type === "image" ? a.image : a.git_url, pill, status: state ?? "" };
    }),
    ...projDatabases.map((d) => ({ type: "db", id: d.id, name: d.name, port: d.port, engine: d.engine, version: d.version, status: d.status, pill: { status: "running", pulse: false } })),
    ...projCompose.map((c) => ({ type: "compose", id: c.id, name: c.name, port: 0, source: "compose", image: "Docker Compose", status: c.status, pill: { status: c.status === "running" ? "running" : "stopped", pulse: c.status === "running" } })),
  ].sort((a, b) => a.name.localeCompare(b.name));

  const dbPill = (status: string): string => {
    if (status === "running" || status === "ready") return "running";
    if (status === "creating" || status === "provisioning") return "provisioning";
    if (status === "failed" || status === "error") return "error";
    if (status === "paused") return "paused";
    return status;
  };

  const selectEnv = (id: string) => {
    const env = envs?.find((e) => e.id === id);
    if (!env) return;
    localStorage.setItem(`aether.env.${projectId}`, env.slug);
    navigate({ to: "/projects/$projectId", params: { projectId }, search: { environment: env.slug } });
  };

  const submitModal = async (v: { name: string; description: string; color: string }) => {
    try {
      if (modal.editing) {
        await update.mutateAsync({ environmentID: modal.editing.id, ...v });
        toast("Environment updated");
      } else {
        const env = await create.mutateAsync(v);
        selectEnv(env.id);
      }
      setModal({ open: false, editing: null });
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
    }
  };

  const confirmDeleteEnv = async () => {
    if (!confirmDelete) return;
    try {
      await del.mutateAsync(confirmDelete.id);
      toast("Environment deleted");
      setConfirmDelete(null);
      const remaining = envs?.filter((e) => e.id !== confirmDelete.id);
      if (remaining?.length) selectEnv(remaining.find((e) => e.is_default)?.id ?? remaining[0].id);
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
    }
  };

  if (!project) return <Spinner label="Loading project..." />;

  return (
    <div className="space-y-lg">
      <div className="flex items-center justify-between gap-lg flex-wrap">
        <div className="flex items-center gap-lg min-w-0">
          <div className="min-w-0">
            <h1 className="font-headline-sm text-headline-sm text-on-surface truncate">{project.name}</h1>
            <p className="font-body-sm text-body-sm text-on-surface-variant">{envs?.length ?? 0} environment(s)</p>
          </div>
          <EnvironmentSwitcher
            envs={envs ?? []}
            selected={selected}
            onSelect={selectEnv}
            onCreate={() => setModal({ open: true, editing: null })}
            onEdit={(e) => setModal({ open: true, editing: { id: e.id, name: e.name, description: e.description, color: e.color } })}
            onSetDefault={(id) => setDefault.mutate(id, { onSuccess: () => toast("Default updated") })}
            onDelete={(e) => setConfirmDelete({ id: e.id, name: e.name })}
          />
        </div>
        <ActionsMenu
            hasEnv={!!selected}
            activeEnv={activeEnv}
            onEnvVars={() => setVarsOpen(true)}
            onProjectVars={() => setProjectVarsOpen(true)}
            onCreate={() => setModal({ open: true, editing: null })}
            onRename={() => activeEnv && setModal({ open: true, editing: { id: activeEnv.id, name: activeEnv.name, description: activeEnv.description, color: activeEnv.color } })}
            onSetDefault={() => activeEnv && setDefault.mutate(activeEnv.id, { onSuccess: () => toast("Default updated") })}
            onDelete={() => activeEnv && setConfirmDelete({ id: activeEnv.id, name: activeEnv.name })}
          />
      </div>

      {isLoading ? (
        <Spinner label="Loading environments..." />
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
              <StatusPill status={tone(activeEnv.status)} pulse={activeEnv.status === "syncing"} />
            </div>
            <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md">
              <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-xs">Last deploy</p>
              <p className="font-headline-sm text-headline-sm text-on-surface">{activeEnv.last_deploy || "—"}</p>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">
              Services · {activeEnv.name}
            </h2>
            <Button variant="primary" className="py-1.5" onClick={() => setCreateOpen(true)}>
              <span className="material-symbols-outlined text-[16px]">add</span>
              Create Service
            </Button>
          </div>

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
                const pill = isDb ? dbPill(svc.status) : svc.pill.status;
                const pulse = isDb ? svc.status === "running" : svc.pill.pulse;
                const target = isCompose ? `/compose/${svc.id}` : isDb ? `/databases/${svc.id}` : `/apps/${svc.id}`;
                const sub = isCompose ? "Docker Compose stack" : isDb ? `${(svc as { engine?: string }).engine} ${(svc as { version?: string }).version}` : (svc as { image?: string }).image ?? "";
                return (
                  <Link
                    key={svc.type + svc.id}
                    to={target}
                    className="group flex flex-col items-start gap-sm px-md py-5 rounded-lg bg-surface-container-lowest border border-outline-variant/50 hover:border-primary/50 hover:bg-surface-container-high/40 transition-colors cursor-pointer min-w-0"
                    title={`Open ${svc.name}`}
                  >
                    <span className="flex items-center justify-between w-full min-w-0">
                      <TechIcon name={isCompose ? "docker" : isDb ? (svc as { engine?: string }).engine : (svc as { source?: string }).source === "git" ? "gitlab" : "docker"} size={16} className="text-primary shrink-0" />
                      <StatusPill status={pill} pulse={pulse} />
                    </span>
                    <span className="w-full min-w-0">
                      <span className="block font-body-md text-body-md text-on-surface truncate">{svc.name}</span>
                      <span className="block font-code-md text-code-md text-on-surface-variant/60 truncate">{sub}{!isCompose && ` · :${svc.port}`}</span>
                    </span>
                    <span className="material-symbols-outlined text-[16px] text-on-surface-variant/30 group-hover:text-primary transition-colors shrink-0 self-end">chevron_right</span>
                  </Link>
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

      <ConfirmDialog
        open={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
        onConfirm={confirmDeleteEnv}
        title="Delete environment"
        description={`Delete "${confirmDelete?.name}"? This is only allowed when the environment has no services.`}
        confirmLabel="Delete"
        danger
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

  const item = (icon: string, label: string, onClick: () => void, danger = false, disabled = false) => (
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
      <span className="material-symbols-outlined text-[16px] w-5 flex justify-center shrink-0">{icon}</span>
      {label}
    </button>
  );

  return (
    <div ref={ref} className="relative inline-block">
      <Button
        variant="subtle"
        className="py-1.5"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="font-label-caps text-label-caps uppercase">Actions</span>
        <span className="material-symbols-outlined text-[16px] opacity-70">{open ? "expand_less" : "expand_more"}</span>
      </Button>
      {open && (
        <div className="absolute right-0 top-full mt-2 z-[70] w-64 rounded-xl bg-surface-popover border border-border-subtle shadow-md overflow-hidden p-1.5">
          <p className="px-2.5 pt-2 pb-1 font-label-caps text-label-caps text-on-surface-variant/60 uppercase">
            Environment · {activeEnv?.name ?? "—"}
          </p>
          {item("tune", "Environment Variables", onEnvVars, false, !hasEnv)}
          <p className="px-2.5 pt-3 pb-1 font-label-caps text-label-caps text-on-surface-variant/60 uppercase">Project</p>
          {item("data_object", "Project Variables", onProjectVars)}
          {item("add", "Create Environment", onCreate)}
          <div className="my-1.5 border-t border-border-subtle" />
          {item("edit", "Rename Environment", onRename, false, !hasEnv)}
          {!activeEnv?.is_default && item("star", "Set as Default", onSetDefault, false, !hasEnv)}
          {item("delete", "Delete Environment", onDelete, true, !hasEnv)}
        </div>
      )}
    </div>
  );
}

export const Route = createFileRoute("/_shell/projects/$projectId/")({
  component: ProjectDetail,
});
