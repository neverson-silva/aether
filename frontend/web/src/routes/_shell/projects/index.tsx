import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useQueryClient } from "@tanstack/react-query";
import { Archive, DotsThree, FolderOpen, GitBranch, Plus, TerminalWindow } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import {
  AlertDialog,
  Badge,
  Button,
  Card,
  Dialog,
  DropdownMenu,
  EmptyState,
  Field,
  Input,
  Skeleton,
  Typography,
  useToast,
} from "@aether/design-system";
import { useApps, useDatabases, useProjects } from "../../../hooks";
import { api } from "../../../api/client";

const schema = z.object({
  name: z.string().trim().min(1, "Name is required").max(64, "Maximum 64 characters"),
});

const designIcon = (icon: typeof Plus) => icon as unknown as DesignIcon;

function formatDate(iso: string) {
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(new Date(iso));
}

function Projects() {
  const { data: projects, isLoading } = useProjects();
  const { data: apps } = useApps();
  const { data: databases } = useDatabases();
  const queryClient = useQueryClient();
  const { add } = useToast();
  const [editing, setEditing] = useState<{ id: string; name: string } | null>(null);
  const [newOpen, setNewOpen] = useState(false);
  const appsByProject = (apps ?? []).reduce<Record<string, number>>((counts, app) => ({ ...counts, [app.project_id]: (counts[app.project_id] ?? 0) + 1 }), {});
  const databasesByProject = (databases ?? []).reduce<Record<string, number>>((counts, database) => ({ ...counts, [database.project_id]: (counts[database.project_id] ?? 0) + 1 }), {});
  const form = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { name: "" } });

  const rename = async (values: z.infer<typeof schema>) => {
    if (!editing) return;
    try {
      await api(`/api/v1/projects/${editing.id}`, { method: "PATCH", body: { name: values.name } });
      add({ title: "Project renamed", tone: "success" });
      setEditing(null);
      await queryClient.invalidateQueries({ queryKey: ["projects"] });
    } catch (error) {
      add({ title: "Unable to rename project", description: error instanceof Error ? error.message : "Try again.", tone: "error" });
    }
  };

  const deleteProject = async (project: { id: string; name: string }) => {
    try {
      await api(`/api/v1/projects/${project.id}`, { method: "DELETE" });
      add({ title: "Project deleted", tone: "success" });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["projects"] }),
        queryClient.invalidateQueries({ queryKey: ["apps"] }),
        queryClient.invalidateQueries({ queryKey: ["databases"] }),
      ]);
    } catch (error) {
      add({ title: "Unable to delete project", description: error instanceof Error ? error.message : "Try again.", tone: "error" });
    }
  };

  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-8 p-6 lg:p-8">
      <header className="flex flex-col gap-5 border-b border-border pb-6 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-2">
          <Typography as="p" level="label" tone="primary">Workspace</Typography>
          <Typography as="h1" level="display">Projects</Typography>
          <Typography as="p" level="body" tone="muted">Organize services, environments and delivery workflows by project.</Typography>
        </div>
        <Dialog
          open={newOpen}
          onOpenChange={setNewOpen}
          title="Create project"
          description="Create a workspace boundary for services and environments."
          trigger={<Button icon={designIcon(Plus)}>New project</Button>}
        >
          <Link to="/projects/new" className="flex items-center gap-3 rounded-lg border border-border p-4 transition-colors hover:bg-surface-container">
            <FolderOpen size={24} weight="duotone" className="text-primary" aria-hidden="true" />
            <span>
              <Typography as="span" level="body" weight="semibold">Project workspace</Typography>
              <Typography as="span" level="small" tone="muted">Group applications and environments.</Typography>
            </span>
          </Link>
        </Dialog>
      </header>

      <section className="space-y-4" aria-labelledby="projects-heading">
        <div className="flex items-center gap-3">
          <Typography as="h2" id="projects-heading" level="heading">Your projects</Typography>
          <Badge tone="neutral" size="md">{projects?.length ?? 0}</Badge>
        </div>

        {isLoading ? <Skeleton variant="card" className="h-48" aria-label="Loading projects" /> : null}
        {!isLoading && !projects?.length ? (
          <EmptyState
            icon={designIcon(FolderOpen)}
            title="No projects yet"
            description="Create a project to group applications, databases and environments."
            action={<Button icon={designIcon(Plus)} onClick={() => setNewOpen(true)}>Create project</Button>}
          />
        ) : null}

        {projects?.length ? (
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
            {projects.map((project, index) => {
              const services = (appsByProject[project.id] ?? 0) + (databasesByProject[project.id] ?? 0);
              const isHealthy = index % 2 === 0;
              return (
                <Card key={project.id} as="article" variant="interactive" padding="none" className="overflow-visible">
                  <div className="flex items-start justify-between gap-4 p-5">
                    <Link to="/projects/$projectId" params={{ projectId: project.id }} className="min-w-0 flex-1">
                      <div className="flex items-center gap-3">
                        <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          {isHealthy ? <GitBranch size={22} weight="duotone" /> : <Archive size={22} weight="duotone" />}
                        </span>
                        <div className="min-w-0">
                          <Typography as="h3" level="heading" truncate>{project.name}</Typography>
                          <Typography as="p" level="small" tone="muted" truncate>{project.description || `${services} service${services === 1 ? "" : "s"}`}</Typography>
                        </div>
                      </div>
                    </Link>
                    <DropdownMenu
                      trigger={<button type="button" aria-label={`Actions for ${project.name}`} className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-surface-container hover:text-foreground"><DotsThree size={20} weight="bold" /></button>}
                      items={[
                        { value: "rename", label: "Rename", onSelect: () => { form.reset({ name: project.name }); setEditing(project); } },
                        { value: "delete", label: <AlertDialog trigger={<span className="block w-full">Delete</span>} title="Delete project" description={`This permanently deletes ${project.name} and all resources inside it.`} confirmLabel="Delete everything" cancelLabel="Keep project" onConfirm={() => deleteProject(project)} /> },
                      ]}
                    />
                  </div>
                  <div className="flex items-center gap-2 border-t border-border px-5 py-3 text-body-sm text-muted-foreground">
                    <TerminalWindow size={16} aria-hidden="true" />
                    <span>{services} services</span>
                    <span className="ml-auto">Created {formatDate(project.created_at)}</span>
                  </div>
                </Card>
              );
            })}
          </div>
        ) : null}
      </section>

      <Dialog open={editing !== null} onOpenChange={(open) => { if (!open) setEditing(null); }} title="Rename project" trigger={<span aria-hidden="true" />}>
        <form onSubmit={form.handleSubmit(rename)} className="space-y-6" noValidate>
          <Field label="Project name" error={form.formState.errors.name?.message}>
            <Input autoFocus {...form.register("name")} />
          </Field>
          <div className="flex justify-end gap-3">
            <Button type="button" variant="ghost" onClick={() => setEditing(null)}>Cancel</Button>
            <Button type="submit" loading={form.formState.isSubmitting}>Save changes</Button>
          </div>
        </form>
      </Dialog>
    </main>
  );
}

export const Route = createFileRoute("/_shell/projects/")({ component: Projects });
