import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useCreateWorker,
  useDeleteWorker,
  useWorkerAction,
  useWorkers,
} from "../../../../hooks";
import { Play, Plus, Stop, Tag, TerminalWindow, Trash } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Button, Card, Dialog, Field, Input, useToast } from "@aether/design-system";
const schema = z.object({
  name: z.string().min(1, "Name is required"),
  command: z.string().min(1, "Command is required"),
});

export function Workers({ appID }: { appID: string }) {
  const { data: workers } = useWorkers(appID);
  const createWorker = useCreateWorker(appID);
  const deleteWorker = useDeleteWorker(appID);
  const workerAction = useWorkerAction();
  const { add } = useToast();
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
      add({ title: "Worker started", tone: "success" });
      setOpen(false);
      reset();
    } catch (err) {
      add({ title: "Could not create worker", description: err instanceof Error ? err.message : "Try again later.", tone: "error" });
    }
  };

  return (
    <Card className="mt-lg">
      <div className="flex items-center justify-between mb-md">
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Workers</h2>
        <Button variant="ghost" icon={Plus as unknown as DesignIcon} onClick={() => setOpen(true)}>New worker</Button>
      </div>
      <div className="overflow-x-auto"><table className="w-full min-w-[640px] text-left"><thead><tr className="border-b border-border text-label-caps text-muted-foreground"><th className="px-3 py-3">Name</th><th className="px-3 py-3">Command</th><th className="px-3 py-3">Status</th><th className="px-3 py-3" /></tr></thead><tbody className="divide-y divide-border">{(workers ?? []).map((w) => <tr key={w.id} className="transition-colors hover:bg-surface-container"><td className="px-3 py-3 font-mono text-code-md text-foreground">{w.name}</td><td className="px-3 py-3 font-mono text-code-md text-muted-foreground">{w.command}</td><td className="px-3 py-3"><Badge tone={w.status === "running" ? "success" : "neutral"}>{w.status}</Badge></td><td className="px-3 py-3"><div className="flex justify-end gap-2"> <Button variant="quiet" size="sm" icon={(w.status === "running" ? Stop : Play) as unknown as DesignIcon} onClick={() => workerAction.mutate({ id: w.id, action: w.status === "running" ? "stop" : "start" }, { onSuccess: () => add({ title: w.status === "running" ? "Worker stopped" : "Worker started", tone: "success" }) })}>{w.status === "running" ? "Stop" : "Start"}</Button><Button variant="quiet" size="sm" aria-label={`Delete ${w.name}`} onClick={() => deleteWorker.mutate(w.id)}><Trash size={16} /></Button></div></td></tr>)}</tbody></table></div>
      {(workers ?? []).length === 0 && (
        <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">No workers.</p>
      )}

      <Dialog open={open} onOpenChange={setOpen} title="New worker" trigger={<button type="button" className="hidden" aria-hidden="true" tabIndex={-1} />}>
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" error={errors.name?.message}>
            <Input leadingIcon={Tag as unknown as DesignIcon} placeholder="queue-consumer" {...register("name")} />
          </Field>
          <Field label="Command" error={errors.command?.message}>
            <Input leadingIcon={TerminalWindow as unknown as DesignIcon} placeholder="node worker.js" {...register("command")} />
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
      </Dialog>
    </Card>
  );
}
