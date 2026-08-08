import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useCronJobs, useCreateCronJob, useDeleteCronJob } from "../../../../api/hooks";
import {
  Button,
  Card,
  Field,
  Input,
  Modal,
  StatusPill,
  Table,
  useToast,
} from "../../../../components/ui";
const schema = z.object({
  name: z.string().min(1, "Name is required"),
  schedule: z.string().min(1, "Schedule is required"),
  command: z.string().min(1, "Command is required"),
});

export function CronJobs({ appID }: { appID: string }) {
  const { data: jobs } = useCronJobs(appID);
  const createJob = useCreateCronJob(appID);
  const deleteJob = useDeleteCronJob(appID);
  const { toast } = useToast();
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
      toast("Cron job created");
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create cron job", "error");
    }
  };

  return (
    <Card className="mt-lg">
      <div className="flex items-center justify-between mb-md">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Cron Jobs</h2>
        <Button variant="ghost" onClick={() => setOpen(true)}>
          <span className="material-symbols-outlined text-[16px]">add</span>
          New job
        </Button>
      </div>
      <Table headers={["Name", "Schedule", "Command", "Enabled", ""]}>
        {(jobs ?? []).map((j) => (
          <tr key={j.id} className="hover:bg-surface-container-high transition-colors">
            <td className="px-sm py-2 font-code-md text-code-md text-on-surface">{j.name}</td>
            <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{j.schedule}</td>
            <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{j.command}</td>
            <td className="px-sm py-2">
              <StatusPill status={j.enabled ? "active" : "disabled"} pulse={j.enabled} />
            </td>
            <td className="px-sm py-2 text-right">
              <button
                onClick={() => deleteJob.mutate(j.id, { onSuccess: () => toast("Cron job removed") })}
                className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
              >
                delete
              </button>
            </td>
          </tr>
        ))}
      </Table>
      {(jobs ?? []).length === 0 && (
        <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">No cron jobs.</p>
      )}

      <Modal open={open} onClose={() => setOpen(false)} title="New cron job">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" hint={errors.name?.message}>
            <Input icon="label" placeholder="nightly-cleanup" {...register("name")} />
          </Field>
          <Field label="Schedule (5-field cron)" hint={errors.schedule?.message}>
            <Input icon="schedule" placeholder="0 2 * * *" {...register("schedule")} />
          </Field>
          <Field label="Command" hint={errors.command?.message}>
            <Input icon="terminal" placeholder="sh -c 'echo hi'" {...register("command")} />
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
      </Modal>
    </Card>
  );
}
