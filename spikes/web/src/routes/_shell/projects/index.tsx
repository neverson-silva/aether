import { AppsInline } from "./-components/AppsInline";
import { createFileRoute } from "@tanstack/react-router";
import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useApps, useAppStates, useDatabases, useProjects } from "../../../api/hooks";
import { TechIcon } from "../../../components/TechIcon";
import { AppLink } from "../../../components/ds";
import { api } from "../../../api/client";
import { useQueryClient } from "@tanstack/react-query";
import {
  Button,
  ConfirmDialog,
  EmptyState,
  Field,
  Input,
  Modal,
  StatusPill,
  fmtDate,
  useToast,
} from "../../../components/ui";

const schema = z.object({
  name: z.string("Name is required").trim().min(1, "Name is required").max(64, "Maximum 64 characters"),
});

function Projects() {
  const { data: projects, isLoading } = useProjects();
  const { data: apps } = useApps();
  const { data: states } = useAppStates();
  const { data: databases } = useDatabases();
  const qc = useQueryClient();
  const { toast } = useToast();
  const [editing, setEditing] = useState<{ id: string; name: string } | null>(null);
  const [deleting, setDeleting] = useState<{ id: string; name: string } | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { name: "" } });

  const appsByProject = (apps ?? []).reduce<Record<string, number>>((acc, a) => {
    acc[a.project_id] = (acc[a.project_id] ?? 0) + 1;
    return acc;
  }, {});
  const dbsByProject = (databases ?? []).reduce<Record<string, number>>((acc, d) => {
    acc[d.project_id] = (acc[d.project_id] ?? 0) + 1;
    return acc;
  }, {});
  const servicesByProject = (id: string) => (appsByProject[id] ?? 0) + (dbsByProject[id] ?? 0);

  const appsOf = (projectId: string) => (apps ?? []).filter((a) => a.project_id === projectId);
  const dbsOf = (projectId: string) => (databases ?? []).filter((d) => d.project_id === projectId);
  const svcOf = (projectId: string) => [
    ...appsOf(projectId).map((a) => ({ type: "app", id: a.id, name: a.name, port: a.port, source: a.source_type, engine: "", version: "", status: "" })),
    ...dbsOf(projectId).map((d) => ({ type: "db", id: d.id, name: d.name, port: d.port, source: "", engine: d.engine, version: d.version, status: d.status })),
  ].sort((a, b) => a.name.localeCompare(b.name));

  const pillFor = (state?: string): { status: string; pulse: boolean } => {
    switch (state) {
      case "running":
        return { status: "running", pulse: true };
      case "paused":
        return { status: "paused", pulse: false };
      case "restarting":
      case "created":
      case "exited":
        return { status: "restarting", pulse: true };
      case "dead":
      case "error":
        return { status: "error", pulse: false };
      default:
        return { status: "provisioning", pulse: true };
    }
  };

  const submitRename = async (values: z.infer<typeof schema>) => {
    if (!editing) return;
    try {
      await api(`/api/v1/projects/${editing.id}`, { method: "PATCH", body: { name: values.name } });
      toast("Project renamed");
      setEditing(null);
      qc.invalidateQueries({ queryKey: ["projects"] });
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to rename", "error");
    }
  };

  const confirmDelete = async () => {
    if (!deleting) return;
    try {
      await api(`/api/v1/projects/${deleting.id}`, { method: "DELETE" });
      toast("Project deleted");
      setDeleting(null);
      qc.invalidateQueries({ queryKey: ["projects"] });
      qc.invalidateQueries({ queryKey: ["apps"] });
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to delete", "error");
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-lg">
        <div>
          <h1 className="font-headline-sm text-headline-sm text-on-surface">Projects</h1>
          <p className="font-body-sm text-body-sm text-on-surface-variant">
            Logical grouping of applications and environments.
          </p>
        </div>
        <AppLink to="/projects/new" variant="primary" size="sm" leftIcon="add">
          New Project
        </AppLink>
      </div>

      {isLoading && <div className="py-md" />}
      {!isLoading && !projects?.length && (
        <EmptyState
          icon="folder_open"
          title="No projects yet"
          description="Projects group your applications and environments. Create one to get started."
          action={
            <AppLink to="/projects/new" variant="primary" size="sm" leftIcon="add">
              Create Project
            </AppLink>
          }
        />
      )}

      {!!projects?.length && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-lg">
          {projects.map((p) => (
            <Link
              key={p.id}
              to="/projects/$projectId"
              params={{ projectId: p.id }}
              className="bg-surface-container-low border border-outline-variant rounded-lg p-lg flex flex-col gap-md hover:border-primary/40 transition-colors"
            >
              <div className="flex items-start justify-between">
                <div className="w-10 h-10 rounded-md bg-primary-container/20 border border-primary/20 flex items-center justify-center">
                  <span className="material-symbols-outlined text-primary text-[20px]">folder_open</span>
                </div>
                <div className="flex items-center gap-sm">
                  <button
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setEditing({ id: p.id, name: p.name });
                    }}
                    className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-primary transition-colors"
                    title="Rename"
                  >
                    edit
                  </button>
                  <button
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setDeleting({ id: p.id, name: p.name });
                    }}
                    className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                    title="Delete"
                  >
                    delete
                  </button>
                </div>
              </div>
              <div className="min-w-0">
                <h2 className="font-headline-sm text-headline-sm text-on-surface truncate">{p.name}</h2>
                <p className="font-code-md text-code-md text-on-surface-variant/60">
                  {servicesByProject(p.id)} service(s)
                </p>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-sm">
                {svcOf(p.id).slice(0, 4).map((svc) => {
                  const isDb = svc.type === "db";
                  const pill = isDb
                    ? svc.status === "ready"
                      ? { status: "running", pulse: true }
                      : svc.status === "failed"
                        ? { status: "error", pulse: false }
                        : { status: "provisioning", pulse: true }
                    : pillFor(states?.[svc.id]);
                  const href = isDb ? `/databases/${svc.id}` : `/apps/${svc.id}`;
                  const sub = isDb ? `${svc.engine} ${svc.version}` : svc.port ? `:${svc.port}` : "";
                  return (
                    <Link
                      key={svc.type + svc.id}
                      to={href}
                      className="group flex items-center gap-sm px-sm py-1.5 rounded bg-surface-container-lowest border border-outline-variant/50 min-w-0 hover:border-primary/50 hover:bg-surface-container-high/40 transition-colors cursor-pointer"
                      title={`Open ${svc.name}`}
                    >
                      <TechIcon name={isDb ? svc.engine : svc.source === "git" ? "gitlab" : "docker"} size={14} className="text-on-surface-variant shrink-0" />
                      <span className="font-code-md text-code-md text-on-surface truncate flex-1">{svc.name}</span>
                      {sub && <span className="font-code-md text-code-md text-on-surface-variant/50 shrink-0">{sub}</span>}
                      <span className="material-symbols-outlined text-[14px] text-on-surface-variant/30 group-hover:text-primary transition-colors shrink-0">chevron_right</span>
                      <StatusPill status={pill.status} pulse={pill.pulse} />
                    </Link>
                  );
                })}
                {servicesByProject(p.id) > 4 && (
                  <p className="font-code-md text-code-md text-on-surface-variant/50">+{servicesByProject(p.id) - 4} more</p>
                )}              </div>
              <div className="flex items-center justify-between mt-auto pt-md border-t border-outline-variant/40">
                <span className="font-code-md text-code-md text-on-surface-variant/60">
                  {fmtDate(p.created_at)}
                </span>
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant/40">
                  arrow_forward
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}

      <div className="mt-lg">
        <AppsInline />
      </div>

      <Modal open={editing !== null} onClose={() => setEditing(null)} title="Rename project">
        <form onSubmit={handleSubmit(submitRename)} className="space-y-lg" noValidate>
          <Field label="Project Name" hint={errors.name?.message}>
            <Input icon="folder_open" placeholder="ex: api-gateway" autoFocus {...register("name")} />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setEditing(null)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              Save
            </Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={confirmDelete}
        title="Delete project"
        description={`This will permanently delete "${deleting?.name}" and EVERYTHING inside it: all applications, containers, deployments, domains, environment variables, databases and compose services. This action cannot be undone.`}
        confirmLabel="Delete everything"
        danger
      />
    </div>
  );
}

export const Route = createFileRoute("/_shell/projects/")({
  component: Projects,
});
