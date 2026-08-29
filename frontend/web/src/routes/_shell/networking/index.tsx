import { ServiceDomains } from "./-components/ServiceDomains";
import { createFileRoute } from "@tanstack/react-router";
import { Link } from "@tanstack/react-router";
import { useNetQ, useServices } from "../../../hooks";
import { Badge, Card, EmptyState } from "@aether/design-system";
import { Gauge, ShareNetwork } from "@phosphor-icons/react";

function LatencyBar({ p50, p95 }: { p50: number; p95: number }) {
  const max = Math.max(100, p95 * 2);
  return (
    <div className="w-24">
      <div className="flex items-center gap-xs">
        <div className="flex-1 h-1.5 rounded bg-surface-container-high overflow-hidden">
          <div className="h-full rounded bg-primary" style={{ width: `${Math.min(100, (p50 / max) * 100)}%` }} />
        </div>
        <span className="font-code-md text-code-md text-on-surface-variant">{p50.toFixed(0)}ms</span>
      </div>
      <div className="flex items-center gap-xs mt-xs">
        <div className="flex-1 h-1.5 rounded bg-surface-container-high overflow-hidden">
          <div className="h-full rounded bg-[#f3c220]" style={{ width: `${Math.min(100, (p95 / max) * 100)}%` }} />
        </div>
        <span className="font-code-md text-code-md text-on-surface-variant/60">{p95.toFixed(0)}ms</span>
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

  return (
    <div className="space-y-lg">
      <header><h1 className="text-headline-sm font-semibold text-foreground">Networking</h1><p className="mt-1 text-body-md text-muted-foreground">Domains, HTTPS and certificates per service. The proxy is dynamically configured in memory.</p></header>

      <Card>
        <div className="flex items-center justify-between mb-md">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Network quality</h2>
          <Gauge size={18} className="text-primary" />
        </div>
        <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
          Probing every application every 30s (HEAD request to the container port).
        </p>
        <div className="bg-surface-container-low border border-outline-variant rounded-lg">
          <table className="w-full text-left"><thead><tr className="text-label-caps text-muted-foreground"><th className="px-2 py-2">App</th><th className="px-2 py-2">Address</th><th className="px-2 py-2">p50 / p95</th><th className="px-2 py-2">Uptime</th><th className="px-2 py-2">HTTP/3</th></tr></thead><tbody>
            {useNetQView().map((n) => (
              <tr key={n.service_id} className="hover:bg-surface-container-high transition-colors">
                <td className="px-sm py-2 font-body-md text-body-md text-on-surface"><Link to="/apps/$appId" params={{ appId: n.service_id }}>{n.name}</Link></td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{n.addr}</td>
                <td className="px-sm py-2"><LatencyBar p50={n.p50_ms} p95={n.p95_ms} /></td>
                <td className="px-sm py-2">
                  <span className="font-code-md text-code-md text-on-surface-variant">{n.uptime_pct.toFixed(1)}%</span>
                </td>
                <td className="px-sm py-2">
                  {n.http3 ? (
                    <Badge tone="success">Enabled</Badge>
                  ) : (
                    <span className="font-body-sm text-body-sm text-on-surface-variant">no</span>
                  )}
                </td>
              </tr>
            ))}
            {useNetQView().length === 0 && (
              <tr>
                <td colSpan={5}>
                  <EmptyState title="No probes yet" description="Deploy an app and wait for the first sample." className="border-0" />
                </td>
              </tr>
            )}
          </tbody></table>
        </div>
      </Card>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-lg">
        <Card>
          <div className="flex items-center justify-between mb-md">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Domains per service</h2>
            <ShareNetwork size={18} className="text-primary" />
          </div>
          <div className="space-y-sm">
            {(services ?? []).length === 0 && (
              <p className="font-body-sm text-body-sm text-on-surface-variant">No services with domains.</p>
            )}
            {(services ?? []).map((service) => <ServiceDomains key={service.id} serviceId={service.id} serviceName={service.name} />)}
          </div>
        </Card>

        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Proxy & TLS</h2>
          <div className="space-y-sm">
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">Proxy provider</span>
              <Badge tone="success">Traefik</Badge>
            </div>
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">Certificates</span>
              <Badge tone="success">Let's Encrypt</Badge>
            </div>
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">Dynamic config</span>
              <Badge tone="info">In memory</Badge>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant/70 pt-sm">
              HTTP-01 for simple domains; DNS-01 (wildcard) via DNS provider plugins. The Certificate
              Engine is sovereign — the proxy only consumes references.
            </p>
          </div>
        </Card>
      </div>
    </div>
  );
}


export const Route = createFileRoute('/_shell/networking/')({
  component: Networking,
});
