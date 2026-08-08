import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useCreateGitOps,
  useDeleteGitOps,
  useGitOps,
  useSyncGitOps,
} from "../../../api/hooks";
import {
  Button,
  Card,
  Field,
  Input,
  Modal,
  Select,
  StatusPill,
  Table,
  useToast,
} from "../../../components/ui";
import { AppPage, AppPageHeader } from "../../../components/ds";

const schema = z.object({
  name: z.string().min(1, "Name is required"),
  repo_url: z.string().min(1, "Repository URL is required"),
  branch: z.string().default("main"),
  path: z.string().default("aether.yml"),
  apply_mode: z.enum(["manual", "auto"]),
});

function statusTone(s: string): string {
  return s === "applied" ? "active" : s === "synced" ? "ready" : s === "apply_failed" ? "failed" : "pending";
}

function GitOps() {
  const { data: configs, isLoading } = useGitOps();
  const create = useCreateGitOps();
  const sync = useSyncGitOps();
  const remove = useDeleteGitOps();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<z.input<typeof schema>, any, z.output<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { branch: "main", path: "aether.yml", apply_mode: "manual" },
  });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await create.mutateAsync(values);
      toast("GitOps config created");
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create", "error");
    }
  };

  return (
    <div className="space-y-lg">
      <AppPageHeader
        title="GitOps"
        description="Declarative infrastructure: watch a repository containing aether.yml, detect drift and apply."
        actions={
          <Button onClick={() => setOpen(true)}>
          <span className="material-symbols-outlined text-[16px]">add</span>
          New config
        </Button>
        }
      />

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-lg">
        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">How it works</h2>
          <ol className="space-y-sm font-body-sm text-body-sm text-on-surface-variant list-decimal list-inside">
            <li>Watch polls the repo every 60s</li>
            <li>aether.yml is parsed and compared to the target org</li>
            <li>Drift is reported as added / changed / removed</li>
            <li>In auto mode changes are applied on the next poll</li>
            <li>Each config targets a dedicated org (gitops-&lt;name&gt;)</li>
          </ol>
        </Card>
        <Card className="xl:col-span-2">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Configs</h2>
          <div className="bg-surface-container-low border border-outline-variant rounded-lg">
            <Table headers={["Name", "Repo", "Branch", "Status", "Drift", "Sync", ""]}>
              {isLoading && (
                <tr>
                  <td colSpan={7} className="px-sm py-lg text-center font-body-sm text-body-sm text-on-surface-variant">Loading…</td>
                </tr>
              )}
              {(configs ?? []).map((g) => (
                <tr key={g.id} className="hover:bg-surface-container-high transition-colors">
                  <td className="px-sm py-2 font-body-md text-body-md text-on-surface">{g.name}</td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{g.repo_url}</td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{g.branch}</td>
                  <td className="px-sm py-2"><StatusPill status={statusTone(g.last_status)} /></td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">
                    +{g.drift_added} ~{g.drift_changed} -{g.drift_removed}
                  </td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">
                    {g.last_sync ? new Date(g.last_sync).toLocaleString() : "—"}
                  </td>
                  <td className="px-sm py-2 text-right space-x-sm">
                    <button
                      onClick={() => sync.mutate(g.id, { onSuccess: () => toast("Sync done") })}
                      className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-primary transition-colors"
                      title="Sync now"
                    >
                      refresh
                    </button>
                    <button
                      onClick={() => remove.mutate(g.id)}
                      className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                    >
                      delete
                    </button>
                  </td>
                </tr>
              ))}
            </Table>
            {(configs ?? []).length === 0 && !isLoading && (
              <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">No GitOps configs yet.</p>
            )}
          </div>
        </Card>
      </div>

      <Modal open={open} onClose={() => setOpen(false)} title="New GitOps config">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" hint={errors.name?.message}>
            <Input icon="label" placeholder="prod" {...register("name")} />
          </Field>
          <Field label="Repository URL" hint={errors.repo_url?.message}>
            <Input icon="code" placeholder="git@github.com:org/infra.git" {...register("repo_url")} />
          </Field>
          <div className="grid grid-cols-2 gap-lg">
            <Field label="Branch" hint={errors.branch?.message}>
              <Input icon="fork_right" {...register("branch")} />
            </Field>
            <Field label="File path" hint={errors.path?.message}>
              <Input icon="description" {...register("path")} />
            </Field>
          </div>
          <Field label="Apply mode">
            <Select {...register("apply_mode")}>
              <option value="manual">Manual (review drift, apply via sync)</option>
              <option value="auto">Auto (apply on every poll)</option>
            </Select>
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">Create</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

export const Route = createFileRoute("/_shell/gitops/")({
  component: GitOps,
});
