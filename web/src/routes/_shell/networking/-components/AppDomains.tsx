import { Link } from "@tanstack/react-router";
import { StatusPill, Table, fmtDate } from "../../../../components/ui";
import { useDomains } from "../../../../hooks";

export function AppDomains({ appId, appName }: { appId: string; appName: string }) {
  const { data: domains } = useDomains("apps", appId);
  if (!domains?.length) return null;
  return (
    <div className="border border-outline-variant/60 rounded p-sm">
      <div className="flex items-center justify-between mb-sm">
        <Link
          to="/apps/$appId"
          params={{ appId }}
          className="font-body-md text-body-md text-primary hover:text-primary-fixed-dim transition-colors"
        >
          {appName}
        </Link>
        <span className="font-code-md text-code-md text-on-surface-variant/60">{domains.length} domain(s)</span>
      </div>
      <Table headers={["Host", "HTTPS", "Cert"]}>
        {domains.map((d) => (
          <tr key={d.id}>
            <td className="px-sm py-1.5 font-code-md text-code-md text-on-surface">{d.host}</td>
            <td className="px-sm py-1.5">{d.https ? <StatusPill status="https" /> : <StatusPill status="http" />}</td>
            <td className="px-sm py-1.5">
              {d.https ? <StatusPill status={d.cert_status} /> : <span className="font-code-md text-code-md text-on-surface-variant/50">—</span>}
            </td>
          </tr>
        ))}
      </Table>
      <p className="font-body-sm text-body-sm text-on-surface-variant/60 mt-sm">{fmtDate(domains[0].created_at)}</p>
    </div>
  );
}
