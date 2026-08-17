import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useApps, useDatabases, useProjects } from "../../../hooks";
import { AppLink } from "../../../components/ds";
import { api } from "../../../api/client";
import { useQueryClient } from "@tanstack/react-query";
import {
  Button,
  CardMenu,
  ConfirmDialog,
  EmptyState,
  Field,
  Input,
  Modal,
  fmtDate,
  useToast,
} from "../../../components/ui";

const schema = z.object({
  name: z.string("Name is required").trim().min(1, "Name is required").max(64, "Maximum 64 characters"),
});

function Projects() {
  const { data: projects, isLoading } = useProjects();
  const { data: apps } = useApps();
  const { data: databases } = useDatabases();
  const qc = useQueryClient();
  const { toast } = useToast();
  const [editing, setEditing] = useState<{ id: string; name: string } | null>(null);
  const [deleting, setDeleting] = useState<{ id: string; name: string } | null>(null);
  const [newOpen, setNewOpen] = useState(false);
  const newRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { name: "" } });

  useEffect(() => {
    if (!newOpen) return;
    const onClick = (e: MouseEvent) => {
      if (newRef.current && !newRef.current.contains(e.target as Node)) {
        setNewOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setNewOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [newOpen]);

  const appsByProject = (apps ?? []).reduce<Record<string, number>>((acc, a) => {
    acc[a.project_id] = (acc[a.project_id] ?? 0) + 1;
    return acc;
  }, {});
  const dbsByProject = (databases ?? []).reduce<Record<string, number>>((acc, d) => {
    acc[d.project_id] = (acc[d.project_id] ?? 0) + 1;
    return acc;
  }, {});
  const servicesByProject = (id: string) => (appsByProject[id] ?? 0) + (dbsByProject[id] ?? 0);

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
    <div className="flex flex-col gap-lg max-w-[1280px] w-full mx-auto">
      {/* Header Section */}
      <header className="flex flex-col md:flex-row md:items-end justify-between gap-md border-b border-outline-variant pb-md">
        <div className="flex flex-col gap-xs">
          <h1 className="font-headline-sm text-[32px] leading-[40px] font-bold text-primary tracking-tight">My Projects</h1>
          <div className="flex items-center gap-xs font-label-sm text-label-sm text-on-surface-variant">
            <span className="w-2 h-2 rounded-full bg-success" />
            Status: Healthy
          </div>
        </div>
        <div className="relative" ref={newRef}>
          <Button variant="primary" size="sm" leftIcon="add" rightIcon="expand_more" onClick={() => setNewOpen((v) => !v)}>
            New
          </Button>
          {newOpen && (
            <div className="absolute right-0 top-full mt-2 z-[70] min-w-56 rounded-xl bg-surface-container-high border border-outline-variant shadow-lg p-1 flex flex-col">
              <Link
                to="/projects/new"
                onClick={() => setNewOpen(false)}
                className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-container text-on-surface transition-colors"
              >
                <span className="material-symbols-outlined text-[18px] text-on-surface-variant">folder_open</span>
                <span className="flex flex-col">
                  <span className="font-label-md text-label-md">Project</span>
                  <span className="font-body-sm text-body-sm text-on-surface-variant/70">Group apps and environments</span>
                </span>
              </Link>
            </div>
          )}
        </div>
      </header>

      {/* Projects section */}
      <section className="flex flex-col gap-md">
        <div className="flex items-center gap-sm">
          <h2 className="font-headline-md text-headline-md font-semibold text-primary">Projects</h2>
          <span className="bg-surface-container px-2 py-1 rounded font-label-sm text-label-sm text-on-surface-variant border border-outline-variant">
            {projects?.length ?? 0}
          </span>
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
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-md">
            {projects.map((p, idx) => (
              <div key={p.id} className="relative group">
                <Link
                  to="/projects/$projectId"
                  params={{ projectId: p.id }}
                  className="block bg-surface-container-lowest border border-outline-variant rounded-lg p-lg flex flex-col gap-md hover:border-outline transition-colors"
                >
                  <div className="flex justify-between items-start gap-sm pr-[44px]">
                    <div className="flex flex-col gap-xs min-w-0">
                      <h3 className="font-title-lg text-title-lg font-medium text-primary flex items-center gap-sm truncate">
                        {p.name}
                        <span className={`w-2 h-2 rounded-full shrink-0 ${idx % 2 === 0 ? "bg-success" : "bg-warning"}`} />
                      </h3>
                      <p className="font-body-md text-body-md text-on-surface-variant">
                        {p.description || `${servicesByProject(p.id)} service(s)`}
                      </p>
                    </div>
                    <span className="material-symbols-outlined text-on-surface-variant text-[20px] shrink-0 mt-[2px]">{idx % 2 === 0 ? "hub" : "database_sync"}</span>
                  </div>
                  <div className="flex items-center gap-sm mt-auto pt-4 border-t border-outline-variant">
                    <div className="flex items-center gap-sm font-label-sm text-label-sm text-on-surface-variant">
                      <span className="material-symbols-outlined text-[14px]">terminal</span>
                      <span className="">{fmtDate(p.created_at)}</span>
                    </div>
                  </div>
                </Link>
                <div className="absolute top-[22px] right-[22px] z-20">
                  <CardMenu
                    items={[
                      { label: "Edit", icon: "edit", onClick: () => setEditing({ id: p.id, name: p.name }) },
                      { label: "Delete", icon: "delete", danger: true, onClick: () => setDeleting({ id: p.id, name: p.name }) },
                    ]}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

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
