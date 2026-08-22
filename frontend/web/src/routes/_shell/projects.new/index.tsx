import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { FolderOpen, Plus } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Button, Card, Field, Input, Typography, useToast } from "@aether/design-system";
import { useCreateProject } from "../../../hooks";

const schema = z.object({
  name: z.string().trim().min(1, "Name is required").max(64, "Maximum 64 characters"),
});

type Form = z.infer<typeof schema>;
const designIcon = (icon: typeof Plus) => icon as unknown as DesignIcon;

function ProjectNew() {
  const form = useForm<Form>({ resolver: zodResolver(schema), defaultValues: { name: "" } });
  const createProject = useCreateProject();
  const navigate = useNavigate();
  const { add } = useToast();

  const submit = async (values: Form) => {
    try {
      await createProject.mutateAsync(values.name);
      add({ title: "Project created", tone: "success" });
      await navigate({ to: "/projects" });
    } catch (error) {
      add({ title: "Unable to create project", description: error instanceof Error ? error.message : "Try again.", tone: "error" });
    }
  };

  return (
    <main className="mx-auto flex w-full max-w-2xl flex-col gap-8 p-6 lg:p-8">
      <header className="space-y-2">
        <Typography as="p" level="label" tone="primary">Workspace</Typography>
        <Typography as="h1" level="display">Create project</Typography>
        <Typography as="p" level="body" tone="muted">Projects group applications and environments for the same initiative.</Typography>
      </header>
      <Card as="section" variant="elevated" padding="lg" header={<div className="flex items-center gap-3"><FolderOpen size={22} weight="duotone" className="text-primary" /><Typography as="h2" level="heading">Project details</Typography></div>}>
        <form onSubmit={form.handleSubmit(submit)} className="space-y-6" noValidate>
          <Field label="Project name" error={form.formState.errors.name?.message}>
            <Input placeholder="e.g. api-gateway" autoFocus {...form.register("name")} />
          </Field>
          <div className="flex justify-end gap-3">
            <Button type="button" variant="ghost" onClick={() => navigate({ to: "/projects" })}>Cancel</Button>
            <Button type="submit" icon={designIcon(Plus)} loading={form.formState.isSubmitting}>Create project</Button>
          </div>
        </form>
      </Card>
    </main>
  );
}

export const Route = createFileRoute("/_shell/projects/new/")({ component: ProjectNew });
