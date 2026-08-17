import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";
import { usePolicy, useSavePolicy, usePolicyEvents } from "../../../../hooks";
import { Button, Card, StatusPill, useToast } from "../../../../components/ui";

const schema = z.object({
  enabled: z.boolean(),
  cpu_min: z.coerce.number().min(0.05).max(64),
  cpu_max: z.coerce.number().min(0.05).max(64),
  mem_min_mb: z.coerce.number().int().min(64).max(262144),
  mem_max_mb: z.coerce.number().int().min(64).max(262144),
  scale_up_pct: z.coerce.number().int().min(10).max(99),
  scale_down_pct: z.coerce.number().int().min(1).max(50),
  cooldown_min: z.coerce.number().int().min(1).max(1440),
});

export function Autopilot({ appID }: { appID: string }) {
  const { data: policy, isLoading } = usePolicy(appID);
  const { data: events } = usePolicyEvents(appID);
  const save = useSavePolicy(appID);
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const { register, handleSubmit } = useForm<z.input<typeof schema>, any, z.output<typeof schema>>({
    resolver: zodResolver(schema),
  });

  if (isLoading) return null;

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await save.mutateAsync({ ...(policy ?? {}), ...values, app_id: appID });
      toast("Policy saved");
      setOpen(false);
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
    }
  };

  return (
    <Card>
      <div className="flex items-center justify-between mb-md">
        <div className="flex items-center gap-sm">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Resource Autopilot</h2>
          <StatusPill status={policy?.enabled ? "active" : "disabled"} pulse={policy?.enabled} />
        </div>
        <Button variant="ghost" onClick={() => setOpen(!open)}>
          {open ? "Close" : "Configure"}
        </Button>
      </div>
      {!open ? (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-md">
          <div>
            <div className="font-label-caps text-label-caps text-on-surface-variant uppercase">Scale up &gt;</div>
            <div className="font-code-md text-code-md text-on-surface">{(policy?.scale_up_pct ?? 80)}% mem</div>
          </div>
          <div>
            <div className="font-label-caps text-label-caps text-on-surface-variant uppercase">Scale down &lt;</div>
            <div className="font-code-md text-code-md text-on-surface">{(policy?.scale_down_pct ?? 15)}% mem</div>
          </div>
          <div>
            <div className="font-label-caps text-label-caps text-on-surface-variant uppercase">Limits</div>
            <div className="font-code-md text-code-md text-on-surface">
              {(policy?.mem_min_mb ?? 128)}–{(policy?.mem_max_mb ?? 2048)} MiB
            </div>
          </div>
          <div>
            <div className="font-label-caps text-label-caps text-on-surface-variant uppercase">CPU</div>
            <div className="font-code-md text-code-md text-on-surface">{(policy?.cpu_min ?? 0.25)}–{(policy?.cpu_max ?? 4)}</div>
          </div>
          {!!events?.length && (
            <div className="col-span-2 md:col-span-4">
              <div className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Recent actions</div>
              <div className="space-y-xs">
                {events.slice(0, 5).map((e) => (
                  <div key={e.id} className="flex items-center justify-between p-xs rounded border border-outline-variant/40">
                    <span className="font-code-md text-code-md text-primary">{e.action}</span>
                    <span className="font-body-sm text-body-sm text-on-surface-variant">{e.detail}</span>
                    <span className="font-code-md text-code-md text-on-surface-variant/60">{new Date(e.created_at).toLocaleString()}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      ) : (
        <form onSubmit={handleSubmit(submit)} className="grid grid-cols-2 md:grid-cols-4 gap-lg">
          <label className="flex items-center gap-sm col-span-4 cursor-pointer select-none">
            <input type="checkbox" defaultChecked={policy?.enabled} className="w-4 h-4 rounded-sm bg-surface border-outline-variant text-primary" {...register("enabled")} />
            <span className="font-body-md text-body-md text-on-surface">Enable autopilot (checks every 60s)</span>
          </label>
          <NumberField label="CPU min" {...register("cpu_min")} defaultValue={policy?.cpu_min} />
          <NumberField label="CPU max" {...register("cpu_max")} defaultValue={policy?.cpu_max} />
          <NumberField label="Mem min (MiB)" {...register("mem_min_mb")} defaultValue={policy?.mem_min_mb} />
          <NumberField label="Mem max (MiB)" {...register("mem_max_mb")} defaultValue={policy?.mem_max_mb} />
          <NumberField label="Scale up above %" {...register("scale_up_pct")} defaultValue={policy?.scale_up_pct} />
          <NumberField label="Scale down below %" {...register("scale_down_pct")} defaultValue={policy?.scale_down_pct} />
          <NumberField label="Cooldown (min)" {...register("cooldown_min")} defaultValue={policy?.cooldown_min} />
          <div className="col-span-4 flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">Save policy</Button>
          </div>
        </form>
      )}
    </Card>
  );
}

function NumberField(props: React.InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  const { label, ...rest } = props;
  return (
    <div>
      <label className="font-label-caps text-label-caps text-on-surface-variant uppercase">{label}</label>
      <input
        type="number"
        step="any"
        className="mt-xs w-full bg-surface-container-low border border-outline-variant rounded-md px-sm py-2 font-code-md text-code-md text-on-surface"
        {...rest}
      />
    </div>
  );
}
