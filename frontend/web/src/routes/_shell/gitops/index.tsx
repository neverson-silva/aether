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
} from "../../../hooks";
import { Badge, Button, Card, Dialog, EmptyState, Field, Input, NativeSelect, Skeleton, useToast } from "@aether/design-system";
import { ArrowsClockwise, Plus, Trash } from "@phosphor-icons/react";

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
  const { add } = useToast();
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
      setOpen(false);
      reset();
    } catch (err) {
      add({ title: "GitOps creation failed", description: err instanceof Error ? err.message : "Unable to create configuration.", tone: "error" });
    }
  };

  return (
    <div className="space-y-lg">
      <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between"><div><h1 className="text-headline-lg text-foreground">GitOps</h1><p className="mt-1 text-body-md text-muted-foreground">Declarative infrastructure: watch a repository containing aether.yml, detect drift and apply.</p></div><Button onClick={() => setOpen(true)}><Plus size={16} />New config</Button></div>

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
          <div className="overflow-x-auto rounded-lg border border-outline-variant bg-surface-container-low"><table className="w-full text-left"><thead><tr className="border-b border-outline-variant text-label-caps text-on-surface-variant"><th className="px-sm py-2">Name</th><th className="px-sm py-2">Repo</th><th className="px-sm py-2">Branch</th><th className="px-sm py-2">Status</th><th className="px-sm py-2">Drift</th><th className="px-sm py-2">Sync</th><th /></tr></thead><tbody>
              {isLoading && (
                <tr>
                  <td colSpan={7} className="px-sm py-sm"><Skeleton variant="table" aria-label="Loading GitOps configurations" /></td>
                </tr>
              )}
              {(configs ?? []).map((g) => (
                <tr key={g.id} className="hover:bg-surface-container-high transition-colors">
                  <td className="px-sm py-2 font-body-md text-body-md text-on-surface">{g.name}</td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{g.repo_url}</td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{g.branch}</td>
                  <td className="px-sm py-2"><Badge tone={g.last_status === "apply_failed" ? "danger" : g.last_status === "applied" ? "success" : "neutral"}>{statusTone(g.last_status)}</Badge></td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">
                    +{g.drift_added} ~{g.drift_changed} -{g.drift_removed}
                  </td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">
                    {g.last_sync ? new Date(g.last_sync).toLocaleString() : "—"}
                  </td>
                  <td className="px-sm py-2 text-right space-x-sm">
                    <button
                      onClick={() => sync.mutate(g.id, { onSuccess: () => add({ title: "Sync complete", tone: "success" }) })}
                      className="text-on-surface-variant hover:text-primary transition-colors"
                      title="Sync now"
                    >
                      <ArrowsClockwise size={16} />
                    </button>
                    <button
                      onClick={() => remove.mutate(g.id)}
                      className="text-on-surface-variant hover:text-error transition-colors"
                    >
                      <Trash size={16} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody></table>
            {(configs ?? []).length === 0 && !isLoading && (
              <EmptyState title="No GitOps configs yet" description="Create a configuration to manage deployments from Git." className="border-0" />
            )}
          </div>
        </Card>
      </div>

      <Dialog open={open} trigger={<span />} onOpenChange={setOpen} title="New GitOps config">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" error={errors.name?.message}>
            <Input placeholder="prod" {...register("name")} />
          </Field>
          <Field label="Repository URL" error={errors.repo_url?.message}>
            <Input placeholder="git@github.com:org/infra.git" {...register("repo_url")} />
          </Field>
          <div className="grid grid-cols-2 gap-lg">
            <Field label="Branch" error={errors.branch?.message}>
              <Input {...register("branch")} />
            </Field>
            <Field label="File path" error={errors.path?.message}>
              <Input {...register("path")} />
            </Field>
          </div>
          <Field label="Apply mode">
            <NativeSelect value="manual" onChange={() => undefined} options={[{ value: "manual", label: "Manual (review drift, apply via sync)" }, { value: "auto", label: "Auto (apply on every poll)" }]} />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">Create</Button>
          </div>
        </form>
      </Dialog>
    </div>
  );
}

export const Route = createFileRoute("/_shell/gitops/")({
  component: GitOps,
});
