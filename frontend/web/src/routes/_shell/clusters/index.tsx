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
import { Badge, Button, Card, Dialog, EmptyState, Field, Input, NativeSelect, useToast } from "@aether/design-system";
import { Cube, LinkBreak, Plus, Trash } from "@phosphor-icons/react";

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
  const { add } = useToast();
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
      add({ title: "Cluster creation failed", description: err instanceof Error ? err.message : "Unable to create cluster.", tone: "error" });
    }
  };

  const unassigned = (servers ?? []).filter((s) => !s.cluster_id || s.cluster_id === "");

  return (
    <div className="space-y-lg">
      <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between"><div><h1 className="text-headline-lg text-foreground">Clusters</h1><p className="mt-1 text-body-md text-muted-foreground">Logical groups of servers. Apps assigned to a cluster deploy only to its nodes.</p></div><Button onClick={() => setOpen(true)}><Plus size={16} />New cluster</Button></div>

      {(clusters ?? []).map((c) => (
        <Card key={c.id}>
          <div className="flex items-center justify-between mb-md">
            <div className="flex items-center gap-sm">
              <Cube size={18} className="text-primary" />
              <h2 className="font-body-md text-body-md text-on-surface">{c.name}</h2>
              <span className="font-code-md text-code-md text-on-surface-variant/60">{c.id}</span>
              {c.labels.map((l) => (
                <span key={l} className="px-1.5 py-0.5 rounded border border-outline-variant font-code-md text-code-md text-primary">{l}</span>
              ))}
            </div>
            <div className="flex items-center gap-sm">
              <Badge tone={c.servers.length > 0 ? "success" : "neutral"} dot>{c.servers.length > 0 ? "Active" : "Disabled"}</Badge>
              <Button variant="ghost" onClick={() => setJoin({ cluster: c.id, name: c.name })}>
                <Plus size={16} />
                Add server
              </Button>
              <button
                onClick={() => remove.mutate(c.id)}
                className="text-on-surface-variant hover:text-error transition-colors"
              >
                <Trash size={16} />
              </button>
            </div>
          </div>
          {c.servers.length === 0 ? (
            <EmptyState title="No servers in this cluster yet" description="Deploys targeting it stay local until a node joins." className="border-0 p-4" />
          ) : (
            <div className="space-y-sm">
              {c.servers.map((srv) => (
                <div key={srv.id} className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
                  <div className="flex items-center gap-sm">
                    <Badge tone={srv.status === "healthy" ? "success" : "neutral"} dot>{srv.status}</Badge>
                    <span className="font-body-md text-body-md text-on-surface">{srv.name}</span>
                    <span className="font-code-md text-code-md text-on-surface-variant/60">{srv.host}</span>
                  </div>
                  <div className="flex items-center gap-md">
                    <span className="font-code-md text-code-md text-on-surface-variant">load {srv.load.toFixed(2)}</span>
                    <button
                      onClick={() => removeServer.mutate({ cluster_id: c.id, server_id: srv.id })}
                      className="text-on-surface-variant hover:text-error transition-colors"
                    >
                      <LinkBreak size={16} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>
      ))}
      {(clusters ?? []).length === 0 && (
        <EmptyState title="No clusters yet" description="Group your agents for affinity-aware scheduling." action={<Button icon={Plus} onClick={() => setOpen(true)}>Create cluster</Button>} />
      )}

      <Dialog open={open} trigger={<span />} onOpenChange={setOpen} title="New cluster">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <Field label="Name" error={errors.name?.message}>
            <Input placeholder="eu-west" {...register("name")} />
          </Field>
          <Field label="Labels (comma separated)" error={errors.labels?.message}>
            <Input placeholder="gpu,ssd" {...register("labels")} />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
            <Button type="submit">Create</Button>
          </div>
        </form>
      </Dialog>

      <Dialog open={join !== null} trigger={<span />} onOpenChange={(value) => { if (!value) setJoin(null); }} title={`Add server to ${join?.name ?? ""}`}>
        {join && (
          <div className="space-y-lg">
            <Field label="Server">
              <NativeSelect
                value=""
                onChange={(e) => {
                  addServer.mutate(
                    { cluster_id: join.cluster, server_id: e.target.value },
                    { onSuccess: () => { add({ title: "Server added", tone: "success" }); setJoin(null); } }
                  );
                }}
                options={[{ value: "", label: "Choose a healthy server…", disabled: true }, ...unassigned.map((s) => ({ value: s.id, label: `${s.name} (${s.status})` }))]}
              />
            </Field>
            {unassigned.length === 0 && (
              <EmptyState title="No unassigned servers available" className="border-0 p-4" />
            )}
          </div>
        )}
      </Dialog>
    </div>
  );
}

export const Route = createFileRoute("/_shell/clusters/")({
  component: Clusters,
});
