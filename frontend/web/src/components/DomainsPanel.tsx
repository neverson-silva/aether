import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useAddDomain,
  useDomains,
  useGenerateFreeDomain,
  useHostInfo,
  useRemoveDomain,
} from "@/hooks";
import { Check, Globe, MagicWand, X } from "@phosphor-icons/react";
import { Badge, Button, Card, Checkbox, Input, useToast } from "@aether/design-system";
import type { Icon as DesignIcon } from "@aether/design-system";

type Kind = "apps" | "databases" | "compose";

const schema = z.object({
  host: z.string().trim().min(1, "Enter a host"),
  https: z.boolean(),
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
  const removeDomain = useRemoveDomain(kind, id);
  const generateFreeDomain = useGenerateFreeDomain(kind, id);
  const form = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { https: true } });

  const submit = form.handleSubmit((values) => {
    addDomain.mutate(values, { onSuccess: () => add({ title: "Domain linked", tone: "success" }), onError: () => add({ title: "Failed to link domain", tone: "error" }) });
  });

  return (
    <div className="space-y-lg">
      <Card>
        <div className="flex items-center justify-between mb-md">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Domains</h2>
          <button
            onClick={() => generateFreeDomain.mutate(undefined, { onSuccess: () => add({ title: "Free domain generated", tone: "success" }) })}
            className="text-primary font-body-sm text-body-sm hover:text-primary-fixed-dim transition-colors flex items-center gap-1"
          >
            <MagicWand size={16} />
            Generate Free Domain
          </button>
        </div>
        <form onSubmit={submit} className="space-y-sm mb-md" noValidate>
          <div className="space-y-xs">
            {form.formState.errors.host && (
              <p className="font-body-sm text-body-sm text-error">{form.formState.errors.host.message}</p>
            )}
            <Input leadingIcon={Globe as unknown as DesignIcon} placeholder="app.example.com" {...form.register("host")} />
          </div>
          <div className="flex items-center gap-md">
            <label className="flex items-center gap-sm cursor-pointer select-none flex-1">
              <Checkbox label="HTTPS (Let's Encrypt)" checked={form.watch("https")} onCheckedChange={(checked) => form.setValue("https", checked === true)} />
            </label>
            <Button type="submit" disabled={addDomain.isPending}>Add</Button>
          </div>
        </form>
        <div className="space-y-sm">
          {(domains ?? []).map((d) => (
            <div key={d.id} className="flex items-center justify-between gap-sm p-sm rounded border border-outline-variant/60">
              <div className="min-w-0">
                <p className="font-code-md text-code-md text-on-surface truncate">{d.host}</p>
                <div className="flex items-center gap-sm mt-xs flex-wrap">
                  <Badge tone={domainPill(d.status) === "active" ? "success" : domainPill(d.status) === "error" ? "danger" : domainPill(d.status) === "pending" ? "warning" : "neutral"}>{d.status}</Badge>
                  {d.https && <span className="font-code-md text-[10px] text-on-surface-variant/60">SSL · {d.cert_status || "requested"}</span>}
                  <span className="font-code-md text-[10px] text-on-surface-variant/60">Port · {d.container_port}</span>
                </div>
              </div>
              <button
                onClick={() => removeDomain.mutate(d.host)}
                aria-label={`Remove ${d.host}`}
                className="text-muted-foreground hover:text-status-danger transition-colors shrink-0"
              >
                <X size={16} />
              </button>
            </div>
          ))}
          {(domains ?? []).length === 0 && (
            <p className="font-body-sm text-body-sm text-on-surface-variant">No domains linked.</p>
          )}
        </div>
      </Card>
      <Card>
        <div className="flex items-center gap-sm mb-sm">
          <Globe size={16} className="text-muted-foreground" />
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Free DNS</h2>
        </div>
        {hostInfo?.free_domain_base ? (
          <div className="space-y-sm">
            <p className="font-body-sm text-body-sm text-on-surface-variant">
              Free subdomains are generated under <span className="font-code-md text-code-md text-on-surface">{hostInfo.free_domain_base}</span> and
              routed by the Traefik ingress on the public interface.
            </p>
            {/\.(nip\.io|sslip\.io|traefik\.me|ngrok-free\.app|ngrok\.app|ngrok\.io)$/i.test(hostInfo.free_domain_base) ? (
              <>
                <div className="rounded border border-outline-variant/60 p-sm font-code-md text-code-md text-on-surface">
                  {hostInfo.public_ip || "your-public-ip"} · wildcard DNS (auto)
                </div>
                <p className="font-body-sm text-body-sm text-on-surface-variant">
                  This base resolves any subdomain to your public IP automatically — no DNS record needed. With HTTPS enabled,
                  Let's Encrypt issues a certificate per subdomain. Generate a free domain when you are ready to expose this service.
                </p>
              </>
            ) : (
              <>
                <div className="rounded border border-outline-variant/60 p-sm font-code-md text-code-md text-on-surface">
                  <span className="text-on-surface-variant">*.</span>{hostInfo.free_domain_base}{" "}
                  <span className="text-on-surface-variant">A →</span> {hostInfo.public_ip || "your-public-ip"}
                </div>
                <p className="font-body-sm text-body-sm text-on-surface-variant">
                  Point a wildcard record <span className="font-code-md text-code-md text-on-surface">*.{hostInfo.free_domain_base}</span> at your public IP, then Let's Encrypt
                  issues a certificate per subdomain. Generate a free domain when you are ready to expose this service.
                </p>
              </>
            )}
          </div>
        ) : (
          <p className="font-body-sm text-body-sm text-on-surface-variant">
            No free-domain base is configured. Point any A or CNAME record you own at the host's public IP
            ({hostInfo?.public_ip || "your-public-ip"}) and link it manually above.
          </p>
        )}
      </Card>
    </div>
  );
}
