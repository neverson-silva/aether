import { createFileRoute } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useNavigate } from "@tanstack/react-router";
import { useCreateProject } from "../../../api/hooks";
import { Button, Field, Input, useToast } from "../../../components/ui";

const schema = z.object({
  name: z.string("Name is required").trim().min(1, "Name is required").max(64, "Maximum 64 characters"),
});

type Form = z.infer<typeof schema>;

function ProjectNew() {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Form>({ resolver: zodResolver(schema), defaultValues: { name: "" } });
  const createProject = useCreateProject();
  const navigate = useNavigate();
  const { toast } = useToast();

  const submit = async (values: Form) => {
    try {
      const project = await createProject.mutateAsync(values.name);
      toast("Project criado");
      navigate({ to: "/projects" });
      return project;
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create project", "error");
    }
  };

  return (
    <div className="max-w-[560px]">
      <div className="mb-lg">
        <h1 className="font-headline-sm text-headline-sm text-on-surface">Novo Project</h1>
        <p className="font-body-sm text-body-sm text-on-surface-variant">
          Projects group applications and environments of the same initiative.
        </p>
      </div>
      <div className="bg-surface-container-low border border-outline-variant rounded-lg p-xl">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Project Name" hint={errors.name?.message}>
            <Input icon="folder_open" placeholder="ex: api-gateway" autoFocus {...register("name")} />
          </Field>
          <div className="flex justify-end">
            <Button type="submit" disabled={isSubmitting}>
              <span className="material-symbols-outlined text-[16px]">add</span>
              Create Project
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

export const Route = createFileRoute('/_shell/projects/new/')({
  component: ProjectNew,
});
