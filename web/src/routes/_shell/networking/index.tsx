import { AppDomains } from "./-components/AppDomains";
import { createFileRoute } from "@tanstack/react-router";
import { Link } from "@tanstack/react-router";
import { useApps, useDomains, useNetQ } from "../../../hooks";
import {
  Card,
  StatusPill,
  Table,
  fmtDate,
} from "../../../components/ui";
import { AppPageHeader } from "../../../components/ds";

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
  const { data: apps } = useApps();

  return (
    <div className="space-y-lg">
      <AppPageHeader
        title="Networking"
        description="Domains, HTTPS and certificates per application. The proxy is dynamically configured in memory."
      />

      <Card>
        <div className="flex items-center justify-between mb-md">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Network quality</h2>
          <span className="material-symbols-outlined text-[18px] text-primary">speed</span>
        </div>
        <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
          Probing every application every 30s (HEAD request to the container port).
        </p>
        <div className="bg-surface-container-low border border-outline-variant rounded-lg">
          <Table headers={["App", "Address", "p50 / p95", "Uptime", "HTTP/3"]}>
            {useNetQView().map((n) => (
              <tr key={n.app_id} className="hover:bg-surface-container-high transition-colors">
                <td className="px-sm py-2 font-body-md text-body-md text-on-surface">{n.name}</td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{n.addr}</td>
                <td className="px-sm py-2"><LatencyBar p50={n.p50_ms} p95={n.p95_ms} /></td>
                <td className="px-sm py-2">
                  <span className="font-code-md text-code-md text-on-surface-variant">{n.uptime_pct.toFixed(1)}%</span>
                </td>
                <td className="px-sm py-2">
                  {n.http3 ? (
                    <StatusPill status="active" />
                  ) : (
                    <span className="font-body-sm text-body-sm text-on-surface-variant">no</span>
                  )}
                </td>
              </tr>
            ))}
            {useNetQView().length === 0 && (
              <tr>
                <td colSpan={5} className="px-sm py-lg text-center font-body-sm text-body-sm text-on-surface-variant">
                  No probes yet — deploy an app and wait for the first sample.
                </td>
              </tr>
            )}
          </Table>
        </div>
      </Card>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-lg">
        <Card>
          <div className="flex items-center justify-between mb-md">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Domains per application</h2>
            <span className="material-symbols-outlined text-[18px] text-primary">router</span>
          </div>
          <div className="space-y-sm">
            {(apps ?? []).length === 0 && (
              <p className="font-body-sm text-body-sm text-on-surface-variant">No applications with domains.</p>
            )}
            {(apps ?? []).map((app) => <AppDomains key={app.id} appId={app.id} appName={app.name} />)}
          </div>
        </Card>

        <Card>
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Proxy & TLS</h2>
          <div className="space-y-sm">
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">Proxy provider</span>
              <StatusPill status="traefik" />
            </div>
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">Certificados</span>
              <StatusPill status="letsencrypt" />
            </div>
            <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
              <span className="font-body-sm text-body-sm text-on-surface">Dynamic config</span>
              <StatusPill status="in-memory" />
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
