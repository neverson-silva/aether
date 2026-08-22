import { Link } from "@tanstack/react-router";
import { Badge } from "@aether/design-system";
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
        <Badge tone="neutral">{domains.length} domains</Badge>
      </div>
      <table className="w-full text-left"><thead><tr className="text-label-caps text-muted-foreground"><th className="px-2 py-2 font-semibold">Host</th><th className="px-2 py-2 font-semibold">HTTPS</th><th className="px-2 py-2 font-semibold">Certificate</th></tr></thead><tbody>
        {domains.map((d) => (
          <tr key={d.id}>
            <td className="px-sm py-1.5 font-code-md text-code-md text-on-surface">{d.host}</td>
            <td className="px-sm py-1.5"><Badge tone={d.https ? "success" : "neutral"}>{d.https ? "HTTPS" : "HTTP"}</Badge></td>
            <td className="px-sm py-1.5">
              {d.https ? <Badge tone={d.cert_status === "valid" || d.cert_status === "issued" ? "success" : "warning"}>{d.cert_status}</Badge> : <span className="font-code-md text-code-md text-on-surface-variant/50">—</span>}
            </td>
          </tr>
        ))}
      </tbody></table>
      <p className="font-body-sm text-body-sm text-on-surface-variant/60 mt-sm">{new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(new Date(domains[0].created_at))}</p>
    </div>
  );
}
