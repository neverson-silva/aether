import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useAddDomain,
  useDomains,
  useGenerateFreeDomain,
  useHostInfo,
  useRemoveDomain,
  useUpdateDomain,
} from "@/hooks";
import { Globe, MagicWand, PencilSimple, Plus, TrashSimple } from "@phosphor-icons/react";
import { Badge, Button, Card, Checkbox, Input, Modal, useToast } from "@aether/design-system";
import type { Icon as DesignIcon } from "@aether/design-system";
import type { Domain } from "@/api/types";

type Kind = "apps" | "databases" | "compose" | "services";

const schema = z.object({
  host: z.string().trim().min(1, "Enter a host"),
  https: z.boolean(),
  container_port: z.string().optional().refine((value) => !value || (/^\d+$/.test(value) && Number(value) >= 1 && Number(value) <= 65535), "Enter a valid container port"),
});

type FormValues = z.infer<typeof schema>;

function domainPill(status: string): string {
  switch (status) {
    case "ACTIVE":
      return "active";
    case "ERROR":
      return "error";
    case "PROVISIONING":
      return "pending";
    default:
      return "disabled";
  }
}

interface DomainsPanelProps {
  kind: Kind;
  id: string;
}

export function DomainsPanel({ kind, id }: DomainsPanelProps) {
  const { add } = useToast();
  const { data: hostInfo } = useHostInfo();
  const { data: domains } = useDomains(kind, id);
  const addDomain = useAddDomain(kind, id);
  const updateDomain = useUpdateDomain(kind, id);
  const removeDomain = useRemoveDomain(kind, id);
  const generateFreeDomain = useGenerateFreeDomain(kind, id);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingDomain, setEditingDomain] = useState<Domain | null>(null);
  const [deletingDomain, setDeletingDomain] = useState<Domain | null>(null);
  const form = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { https: true, container_port: "" } });

  const openCreate = () => {
    setEditingDomain(null);
    form.reset({ host: "", https: true, container_port: "" });
    setEditorOpen(true);
  };

  const openEdit = (domain: Domain) => {
    setEditingDomain(domain);
    form.reset({ host: domain.host, https: domain.https, container_port: String(domain.container_port) });
    setEditorOpen(true);
  };

  const closeEditor = () => {
    setEditorOpen(false);
    setEditingDomain(null);
  };

  const submit = form.handleSubmit((values) => {
    const body = { ...values, container_port: values.container_port ? Number(values.container_port) : undefined };
    const options = {
      onSuccess: () => {
        closeEditor();
        add({ title: editingDomain ? "Domain updated" : "Domain linked", tone: "success" });
      },
      onError: () => add({ title: editingDomain ? "Failed to update domain" : "Failed to link domain", tone: "error" }),
    };
    if (editingDomain) {
      updateDomain.mutate({ domainID: editingDomain.id, body }, options);
      return;
    }
    addDomain.mutate(body, options);
  });

  const confirmDelete = () => {
    if (!deletingDomain) return;
    removeDomain.mutate(deletingDomain.host, {
      onSuccess: () => {
        setDeletingDomain(null);
        add({ title: "Domain removed", tone: "success" });
      },
      onError: () => add({ title: "Failed to remove domain", tone: "error" }),
    });
  };

  const saving = addDomain.isPending || updateDomain.isPending;

  return (
    <div className="space-y-lg">
      <Card>
        <div className="mb-md flex items-center justify-between gap-md">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Domains</h2>
          <div className="flex items-center gap-md">
            <button onClick={() => generateFreeDomain.mutate(undefined, { onSuccess: () => add({ title: "Free domain generated", tone: "success" }) })} className="flex items-center gap-1 font-body-sm text-body-sm text-primary transition-colors hover:text-primary-fixed-dim">
              <MagicWand size={16} />
              Generate Free Domain
            </button>
            <Button size="sm" icon={Plus} onClick={openCreate}>Add domain</Button>
          </div>
        </div>
        {(domains ?? []).length > 0 ? (
          <div className="grid gap-sm md:grid-cols-2">
            {(domains ?? []).map((domain) => {
              const tone = domainPill(domain.status);
              return (
                <div key={domain.id} className="group relative rounded-xl border border-outline-variant/60 bg-surface-container-low transition-all duration-200 hover:border-primary hover:bg-surface-container">
                  <button type="button" onClick={() => openEdit(domain)} className="w-full rounded-xl p-md pr-24 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary">
                    <div className="flex items-start gap-sm">
                      <Globe size={20} className="mt-0.5 shrink-0 text-primary" />
                      <div className="min-w-0">
                        <p className="truncate font-code-md text-code-md text-on-surface">{domain.host}</p>
                        <div className="mt-sm flex flex-wrap items-center gap-sm">
                          <Badge tone={tone === "active" ? "success" : tone === "error" ? "danger" : tone === "pending" ? "warning" : "neutral"}>{domain.status}</Badge>
                          <span className="font-code-md text-[10px] text-on-surface-variant/70">Port · {domain.container_port}</span>
                          {domain.https && <span className="font-code-md text-[10px] text-on-surface-variant/70">SSL · {domain.cert_status || "requested"}</span>}
                        </div>
                      </div>
                    </div>
                  </button>
                  <div className="absolute right-md top-md flex items-center gap-xs opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
                    <button type="button" onClick={() => openEdit(domain)} aria-label={`Edit ${domain.host}`} title="Edit domain" className="inline-flex size-8 items-center justify-center rounded-md text-on-surface-variant hover:bg-surface-container-high hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary">
                      <PencilSimple size={16} />
                    </button>
                    <button type="button" onClick={() => setDeletingDomain(domain)} aria-label={`Remove ${domain.host}`} title="Remove domain" className="inline-flex size-8 items-center justify-center rounded-md text-on-surface-variant hover:bg-status-danger-container/20 hover:text-status-danger focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-status-danger">
                      <TrashSimple size={16} />
                    </button>
                  </div>
                  <span className="pointer-events-none absolute bottom-md right-md text-[10px] text-on-surface-variant/60 opacity-0 transition-opacity group-hover:opacity-100">Click to edit</span>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="rounded-xl border border-dashed border-outline-variant/70 bg-surface-container-low p-lg text-center">
            <Globe size={24} className="mx-auto mb-sm text-primary" />
            <p className="font-body-md text-body-md text-on-surface">No domains linked</p>
            <p className="mt-xs font-body-sm text-body-sm text-on-surface-variant">Add a domain to route traffic to this service.</p>
            <Button className="mt-md" size="sm" icon={Plus} onClick={openCreate}>Add domain</Button>
          </div>
        )}
      </Card>

      <Modal open={editorOpen} onOpenChange={(open) => !open && closeEditor()} title={editingDomain ? "Edit domain" : "Add domain"} description={editingDomain ? "Update routing and TLS settings for this domain." : "Connect a domain to this service."} size="sm">
        <form onSubmit={submit} className="space-y-md" noValidate>
          <div className="space-y-xs">
            <p className="font-body-sm text-body-sm text-on-surface-variant">Domain</p>
            <Input leadingIcon={Globe as unknown as DesignIcon} placeholder="app.example.com" {...form.register("host")} />
            {form.formState.errors.host && <p className="font-body-sm text-body-sm text-error">{form.formState.errors.host.message}</p>}
          </div>
          <div className="space-y-xs">
            <p className="font-body-sm text-body-sm text-on-surface-variant">Container port</p>
            <Input type="number" min={1} max={65535} placeholder="3000" {...form.register("container_port")} />
            {form.formState.errors.container_port && <p className="font-body-sm text-body-sm text-error">{form.formState.errors.container_port.message}</p>}
          </div>
          <label className="flex cursor-pointer select-none items-center gap-sm">
            <Checkbox label="HTTPS (Let's Encrypt)" checked={form.watch("https")} onCheckedChange={(checked) => form.setValue("https", checked === true)} />
          </label>
          <div className="flex justify-end gap-sm border-t border-outline-variant pt-md">
            <Button type="button" variant="ghost" onClick={closeEditor}>Cancel</Button>
            <Button type="submit" loading={saving}>{editingDomain ? "Save changes" : "Add domain"}</Button>
          </div>
        </form>
      </Modal>

      <Modal open={Boolean(deletingDomain)} onOpenChange={(open) => !open && setDeletingDomain(null)} title="Remove domain" description="This will remove the domain routing configuration." size="sm">
        <div className="space-y-md">
          <p className="font-body-md text-body-md text-on-surface">Remove <span className="font-code-md text-code-md">{deletingDomain?.host}</span> from this service?</p>
          <div className="flex justify-end gap-sm border-t border-outline-variant pt-md">
            <Button variant="ghost" onClick={() => setDeletingDomain(null)}>Cancel</Button>
            <Button variant="danger" loading={removeDomain.isPending} onClick={confirmDelete}>Remove domain</Button>
          </div>
        </div>
      </Modal>

      <Card>
        <div className="mb-sm flex items-center gap-sm">
          <Globe size={16} className="text-muted-foreground" />
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Free DNS</h2>
        </div>
        {hostInfo?.free_domain_base ? (
          <div className="space-y-sm">
            <p className="font-body-sm text-body-sm text-on-surface-variant">Free subdomains are generated under <span className="font-code-md text-code-md text-on-surface">{hostInfo.free_domain_base}</span> and routed by the Traefik ingress on the public interface.</p>
            {/\.(nip\.io|sslip\.io|traefik\.me|ngrok-free\.app|ngrok\.app|ngrok\.io)$/i.test(hostInfo.free_domain_base) ? (
              <>
                <div className="rounded border border-outline-variant/60 p-sm font-code-md text-code-md text-on-surface">{hostInfo.public_ip || "your-public-ip"} · wildcard DNS (auto)</div>
                <p className="font-body-sm text-body-sm text-on-surface-variant">This base resolves any subdomain to your public IP automatically — no DNS record needed. With HTTPS enabled, Let's Encrypt issues a certificate per subdomain. Generate a free domain when you are ready to expose this service.</p>
              </>
            ) : (
              <>
                <div className="rounded border border-outline-variant/60 p-sm font-code-md text-code-md text-on-surface"><span className="text-on-surface-variant">*.</span>{hostInfo.free_domain_base} <span className="text-on-surface-variant">A →</span> {hostInfo.public_ip || "your-public-ip"}</div>
                <p className="font-body-sm text-body-sm text-on-surface-variant">Point a wildcard record <span className="font-code-md text-code-md text-on-surface">*.{hostInfo.free_domain_base}</span> at your public IP, then Let's Encrypt issues a certificate per subdomain. Generate a free domain when you are ready to expose this service.</p>
              </>
            )}
          </div>
        ) : (
          <p className="font-body-sm text-body-sm text-on-surface-variant">No free-domain base is configured. Point any A or CNAME record you own at the host's public IP ({hostInfo?.public_ip || "your-public-ip"}) and link it manually above.</p>
        )}
      </Card>
    </div>
  );
}
