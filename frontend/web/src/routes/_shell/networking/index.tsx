import { ServiceDomains } from "./-components/ServiceDomains";
import { createFileRoute } from "@tanstack/react-router";
import { Link } from "@tanstack/react-router";
import { useNetQ, useServices } from "../../../hooks";
import { useSaveServerDomains, useServerDomains } from "../../../hooks";
import { Badge, Button, Card, Checkbox, EmptyState, Field, Input, InlineError, useToast } from "@aether/design-system";
import { Gauge, Globe, ShareNetwork } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { PageHeader } from "../../../components/PageHeader";

function LatencyBar({ p50, p95 }: { p50: number; p95: number }) {
  const max = Math.max(100, p95 * 2);
  return (
    <div className="w-24">
      <div className="flex items-center gap-xs">
        <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-container-high">
          <div
            className="h-full rounded-full bg-primary"
            style={{ width: `${Math.min(100, (p50 / max) * 100)}%` }}
          />
        </div>
        <span className="font-code-md text-code-md text-on-surface-variant">
          {p50.toFixed(0)}ms
        </span>
      </div>
      <div className="flex items-center gap-xs mt-xs">
        <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-container-high">
          <div
            className="h-full rounded-full bg-status-warning"
            style={{ width: `${Math.min(100, (p95 / max) * 100)}%` }}
          />
        </div>
        <span className="font-code-md text-code-md text-on-surface-variant/60">
          {p95.toFixed(0)}ms
        </span>
      </div>
    </div>
  );
}

function useNetQView() {
  const { data } = useNetQ();
  return data ?? [];
}

function Networking() {
  const { data: services } = useServices();
  const serverDomains = useServerDomains();
  const saveServerDomains = useSaveServerDomains();
  const { add } = useToast();
  const [webDomain, setWebDomain] = useState("");
  const [apiDomain, setApiDomain] = useState("");
  const [https, setHttps] = useState(true);

  useEffect(() => {
    if (!serverDomains.data) return;
    setWebDomain(serverDomains.data.web_domain);
    setApiDomain(serverDomains.data.api_domain);
    setHttps(serverDomains.data.https);
  }, [serverDomains.data]);

  const save = () => {
    saveServerDomains.mutate(
      { web_domain: webDomain.trim(), api_domain: apiDomain.trim(), https },
      {
        onSuccess: () => add({ title: "Aether domains saved", description: "Traefik is updating the routes and certificates.", tone: "success" }),
        onError: (error) => add({ title: "Could not save Aether domains", description: error.message, tone: "error" }),
      },
    );
  };

  return (
    <div className="space-y-lg">
      <PageHeader
        eyebrow="Connectivity"
        title="Networking"
        description="Domains, HTTPS and certificates per service. The proxy is dynamically configured in memory."
      />

      <Card>
        <div className="flex items-center justify-between gap-md mb-md">
          <div>
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Aether server domains</h2>
            <p className="mt-xs font-body-sm text-body-sm text-on-surface-variant">Expose the Aether web interface and API through Traefik.</p>
          </div>
          <Globe size={20} className="text-primary" />
        </div>
        {serverDomains.isError ? <InlineError message="Only the global administrator can manage server domains." /> : null}
        <div className="grid grid-cols-1 gap-md lg:grid-cols-2">
          <Field label="Web domain">
            <Input value={webDomain} onChange={(event) => setWebDomain(event.target.value)} placeholder="aether.example.com" />
          </Field>
          <Field label="API domain">
            <Input value={apiDomain} onChange={(event) => setApiDomain(event.target.value)} placeholder="api.aether.example.com" />
          </Field>
        </div>
        <div className="mt-md flex flex-wrap items-center justify-between gap-md border-t border-outline-variant pt-md">
          <Checkbox label="HTTPS (Let's Encrypt)" checked={https} onCheckedChange={(checked) => setHttps(checked === true)} />
          <Button onClick={save} loading={saveServerDomains.isPending}>Save domains</Button>
        </div>
      </Card>

      <Card>
        <div className="flex items-center justify-between mb-md">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">
            Network quality
          </h2>
          <Gauge size={18} className="text-primary" />
        </div>
        <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
          Probing every application every 30s (HEAD request to the container
          port).
        </p>
        <div className="overflow-hidden rounded-2xl border border-outline-variant bg-surface-container-low">
          <table className="w-full text-left">
            <thead>
              <tr className="text-label-caps text-muted-foreground">
                <th className="px-2 py-2">App</th>
                <th className="px-2 py-2">Address</th>
                <th className="px-2 py-2">p50 / p95</th>
                <th className="px-2 py-2">Uptime</th>
                <th className="px-2 py-2">HTTP/3</th>
              </tr>
            </thead>
            <tbody>
              {useNetQView().map((n) => (
                <tr
                  key={n.service_id}
                  className="hover:bg-surface-container-high transition-colors"
                >
                  <td className="px-sm py-2 font-body-md text-body-md text-on-surface">
                    <Link to="/apps/$appId" params={{ appId: n.service_id }}>
                      {n.name}
                    </Link>
                  </td>
                  <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">
                    {n.addr}
                  </td>
                  <td className="px-sm py-2">
                    <LatencyBar p50={n.p50_ms} p95={n.p95_ms} />
                  </td>
                  <td className="px-sm py-2">
                    <span className="font-code-md text-code-md text-on-surface-variant">
                      {n.uptime_pct.toFixed(1)}%
                    </span>
                  </td>
                  <td className="px-sm py-2">
                    {n.http3 ? (
                      <Badge tone="success">Enabled</Badge>
                    ) : (
                      <span className="font-body-sm text-body-sm text-on-surface-variant">
                        no
                      </span>
                    )}
                  </td>
                </tr>
              ))}
              {useNetQView().length === 0 && (
                <tr>
                  <td colSpan={5}>
                    <EmptyState
                      title="No probes yet"
                      description="Deploy an app and wait for the first sample."
                      className="border-0"
                    />
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-lg">
        <Card>
          <div className="flex items-center justify-between mb-md">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">
              Domains per service
            </h2>
            <ShareNetwork size={18} className="text-primary" />
          </div>
          <div className="space-y-sm">
            {(services ?? []).length === 0 && (
              <p className="font-body-sm text-body-sm text-on-surface-variant">
                No services with domains.
              </p>
            )}
            {(services ?? []).map((service) => (
              <ServiceDomains
                key={service.id}
                serviceId={service.id}
                serviceName={service.name}
              />
            ))}
          </div>
        </Card>

        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
            Proxy & TLS
          </h2>
          <div className="space-y-sm">
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">
                Proxy provider
              </span>
              <Badge tone="success">Traefik</Badge>
            </div>
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">
                Certificates
              </span>
              <Badge tone="success">Let's Encrypt</Badge>
            </div>
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">
                Dynamic config
              </span>
              <Badge tone="info">In memory</Badge>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant/70 pt-sm">
              HTTP-01 for simple domains; DNS-01 (wildcard) via DNS provider
              plugins. The Certificate Engine is sovereign — the proxy only
              consumes references.
            </p>
          </div>
        </Card>
      </div>
    </div>
  );
}

export const Route = createFileRoute("/_shell/networking/")({
  component: Networking,
});
