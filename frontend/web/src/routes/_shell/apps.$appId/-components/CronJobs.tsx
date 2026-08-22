import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useCronJobs, useCreateCronJob, useDeleteCronJob } from "../../../../hooks";
import { Clock, Plus, TerminalWindow, Tag, Trash } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Button, Card, Dialog, Field, Input, useToast } from "@aether/design-system";
const schema = z.object({
  name: z.string().min(1, "Name is required"),
  schedule: z.string().min(1, "Schedule is required"),
  command: z.string().min(1, "Command is required"),
});

export function CronJobs({ appID }: { appID: string }) {
  const { data: jobs } = useCronJobs(appID);
  const createJob = useCreateCronJob(appID);
  const deleteJob = useDeleteCronJob(appID);
  const { add } = useToast();
  const [open, setOpen] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { schedule: "0 2 * * *" },
  });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await createJob.mutateAsync(values);
      setOpen(false);
      reset();
    } catch (err) {
      add({ title: "Could not create cron job", description: err instanceof Error ? err.message : "Try again later.", tone: "error" });
    }
  };

  return (
    <Card className="mt-lg">
      <div className="flex items-center justify-between mb-md">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Cron Jobs</h2>
        <Button variant="ghost" icon={Plus as unknown as DesignIcon} onClick={() => setOpen(true)}>New job</Button>
      </div>
      <div className="overflow-x-auto"><table className="w-full min-w-[680px] text-left"><thead><tr className="border-b border-border text-label-caps text-muted-foreground"><th className="px-3 py-3">Name</th><th className="px-3 py-3">Schedule</th><th className="px-3 py-3">Command</th><th className="px-3 py-3">Enabled</th><th className="px-3 py-3" /></tr></thead><tbody className="divide-y divide-border">{(jobs ?? []).map((j) => <tr key={j.id} className="transition-colors hover:bg-surface-container"><td className="px-3 py-3 font-mono text-code-md text-foreground">{j.name}</td><td className="px-3 py-3 font-mono text-code-md text-muted-foreground">{j.schedule}</td><td className="px-3 py-3 font-mono text-code-md text-muted-foreground">{j.command}</td><td className="px-3 py-3"><Badge tone={j.enabled ? "success" : "neutral"}>{j.enabled ? "Active" : "Disabled"}</Badge></td><td className="px-3 py-3 text-right"><Button variant="quiet" size="sm" aria-label={`Delete ${j.name}`} onClick={() => deleteJob.mutate(j.id)}><Trash size={16} /></Button></td></tr>)}</tbody></table></div>
      {(jobs ?? []).length === 0 && (
        <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">No cron jobs.</p>
      )}

      <Dialog open={open} onOpenChange={setOpen} title="New cron job" trigger={<button type="button" className="hidden" aria-hidden="true" tabIndex={-1} />}>
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" error={errors.name?.message}>
            <Input leadingIcon={Tag as unknown as DesignIcon} placeholder="nightly-cleanup" {...register("name")} />
          </Field>
          <Field label="Schedule (5-field cron)" error={errors.schedule?.message}>
            <Input leadingIcon={Clock as unknown as DesignIcon} placeholder="0 2 * * *" {...register("schedule")} />
          </Field>
          <Field label="Command" error={errors.command?.message}>
            <Input leadingIcon={TerminalWindow as unknown as DesignIcon} placeholder="sh -c 'echo hi'" {...register("command")} />
          </Field>
          <div className="flex justify-end gap-md">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              Create
            </Button>
          </div>
        </form>
      </Dialog>
    </Card>
  );
}
