import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useCreateWorker,
  useDeleteWorker,
  useWorkerAction,
  useWorkers,
} from "../../../../api/hooks";
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
  command: z.string().min(1, "Command is required"),
});

export function Workers({ appID }: { appID: string }) {
  const { data: workers } = useWorkers(appID);
  const createWorker = useCreateWorker(appID);
  const deleteWorker = useDeleteWorker(appID);
  const workerAction = useWorkerAction();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { name: "" } });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await createWorker.mutateAsync(values);
      toast("Worker started");
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create worker", "error");
    }
  };

  return (
    <Card className="mt-lg">
      <div className="flex items-center justify-between mb-md">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Workers</h2>
        <Button variant="ghost" onClick={() => setOpen(true)}>
          <span className="material-symbols-outlined text-[16px]">add</span>
          New worker
        </Button>
      </div>
      <Table headers={["Name", "Command", "Status", ""]}>
        {(workers ?? []).map((w) => (
          <tr key={w.id} className="hover:bg-surface-container-high transition-colors">
            <td className="px-sm py-2 font-code-md text-code-md text-on-surface">{w.name}</td>
            <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{w.command}</td>
            <td className="px-sm py-2">
              <StatusPill status={w.status} pulse={w.status === "running"} />
            </td>
            <td className="px-sm py-2">
              <div className="flex items-center gap-sm justify-end">
                {w.status === "running" ? (
                  <button
                    onClick={() =>
                      workerAction.mutate({ id: w.id, action: "stop" }, { onSuccess: () => toast("Worker stopped") })
                    }
                    className="font-label-caps text-label-caps text-tertiary hover:text-tertiary-fixed-dim transition-colors uppercase"
                  >
                    Stop
                  </button>
                ) : (
                  <button
                    onClick={() =>
                      workerAction.mutate({ id: w.id, action: "start" }, { onSuccess: () => toast("Worker started") })
                    }
                    className="font-label-caps text-label-caps text-primary hover:text-primary-fixed-dim transition-colors uppercase"
                  >
                    Start
                  </button>
                )}
                <button
                  onClick={() => deleteWorker.mutate(w.id, { onSuccess: () => toast("Worker removed") })}
                  className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                >
                  delete
                </button>
              </div>
            </td>
          </tr>
        ))}
      </Table>
      {(workers ?? []).length === 0 && (
        <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">No workers.</p>
      )}

      <Modal open={open} onClose={() => setOpen(false)} title="New worker">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" hint={errors.name?.message}>
            <Input icon="label" placeholder="queue-consumer" {...register("name")} />
          </Field>
          <Field label="Command" hint={errors.command?.message}>
            <Input icon="terminal" placeholder="node worker.js" {...register("command")} />
          </Field>
          <div className="flex justify-end gap-md">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              Create & start
            </Button>
          </div>
        </form>
      </Modal>
    </Card>
  );
}
