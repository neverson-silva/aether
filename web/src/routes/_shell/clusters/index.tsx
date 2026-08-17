import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useClusterAddServer,
  useClusterRemoveServer,
  useClusters,
  useCreateCluster,
  useDeleteCluster,
  useServers,
} from "../../../hooks";
import {
  Button,
  Card,
  Field,
  Input,
  Modal,
  Select,
  StatusPill,
  useToast,
} from "../../../components/ui";
import { AppPage, AppPageHeader } from "../../../components/ds";

const schema = z.object({
  name: z.string().min(1, "Name is required"),
  labels: z.string().optional(),
});

function Clusters() {
  const { data: clusters } = useClusters();
  const { data: servers } = useServers();
  const create = useCreateCluster();
  const remove = useDeleteCluster();
  const addServer = useClusterAddServer();
  const removeServer = useClusterRemoveServer();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const [join, setJoin] = useState<{ cluster: string; name: string } | null>(null);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<z.input<typeof schema>, any, z.output<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", labels: "" },
  });

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await create.mutateAsync({
        name: values.name,
        labels: (values.labels || "").split(",").map((l) => l.trim()).filter(Boolean),
      });
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
    }
  };

  const unassigned = (servers ?? []).filter((s) => !s.cluster_id || s.cluster_id === "");

  return (
    <div className="space-y-lg">
      <AppPageHeader
        title="Clusters"
        description="Logical groups of servers. Apps assigned to a cluster deploy only to its nodes."
        actions={
          <Button onClick={() => setOpen(true)}>
          <span className="material-symbols-outlined text-[16px]">add</span>
          New cluster
        </Button>
        }
      />

      {(clusters ?? []).map((c) => (
        <Card key={c.id}>
          <div className="flex items-center justify-between mb-md">
            <div className="flex items-center gap-sm">
              <span className="material-symbols-outlined text-[18px] text-primary">hub</span>
              <h2 className="font-body-md text-body-md text-on-surface">{c.name}</h2>
              <span className="font-code-md text-code-md text-on-surface-variant/60">{c.id}</span>
              {c.labels.map((l) => (
                <span key={l} className="px-1.5 py-0.5 rounded border border-outline-variant font-code-md text-code-md text-primary">{l}</span>
              ))}
            </div>
            <div className="flex items-center gap-sm">
              <StatusPill status={c.servers.length > 0 ? "active" : "disabled"} pulse={c.servers.length > 0} />
              <Button variant="ghost" onClick={() => setJoin({ cluster: c.id, name: c.name })}>
                <span className="material-symbols-outlined text-[16px]">add</span>
                Add server
              </Button>
              <button
                onClick={() => remove.mutate(c.id)}
                className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
              >
                delete
              </button>
            </div>
          </div>
          {c.servers.length === 0 ? (
            <p className="font-body-sm text-body-sm text-on-surface-variant">
              No servers in this cluster yet. Deploys targeting it stay local until a node joins.
            </p>
          ) : (
            <div className="space-y-sm">
              {c.servers.map((srv) => (
                <div key={srv.id} className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
                  <div className="flex items-center gap-sm">
                    <StatusPill status={srv.status} pulse={srv.status === "healthy"} />
                    <span className="font-body-md text-body-md text-on-surface">{srv.name}</span>
                    <span className="font-code-md text-code-md text-on-surface-variant/60">{srv.host}</span>
                  </div>
                  <div className="flex items-center gap-md">
                    <span className="font-code-md text-code-md text-on-surface-variant">load {srv.load.toFixed(2)}</span>
                    <button
                      onClick={() => removeServer.mutate({ cluster_id: c.id, server_id: srv.id })}
                      className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                    >
                      link_off
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>
      ))}
      {(clusters ?? []).length === 0 && (
        <Card>
          <p className="font-body-sm text-body-sm text-on-surface-variant">
            No clusters yet. Group your agents for affinity-aware scheduling.
          </p>
        </Card>
      )}

      <Modal open={open} onClose={() => setOpen(false)} title="New cluster">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" hint={errors.name?.message}>
            <Input icon="hub" placeholder="eu-west" {...register("name")} />
          </Field>
          <Field label="Labels (comma separated)" hint={errors.labels?.message}>
            <Input icon="sell" placeholder="gpu,ssd" {...register("labels")} />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">Create</Button>
          </div>
        </form>
      </Modal>

      <Modal open={join !== null} onClose={() => setJoin(null)} title={`Add server to ${join?.name ?? ""}`}>
        {join && (
          <div className="space-y-lg">
            <Field label="Server">
              <Select
                value=""
                onChange={(e) => {
                  addServer.mutate(
                    { cluster_id: join.cluster, server_id: e.target.value },
                    { onSuccess: () => { toast("Server added"); setJoin(null); } }
                  );
                }}
              >
                <option value="" disabled>Choose a healthy server…</option>
                {unassigned.map((s) => (
                  <option key={s.id} value={s.id}>{s.name} ({s.status})</option>
                ))}
              </Select>
            </Field>
            {unassigned.length === 0 && (
              <p className="font-body-sm text-body-sm text-on-surface-variant">
                No unassigned servers available.
              </p>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}

export const Route = createFileRoute("/_shell/clusters/")({
  component: Clusters,
});
