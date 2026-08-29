import { Link } from "@tanstack/react-router";
import { Badge } from "@aether/design-system";
import { useDomains } from "../../../../hooks";

export function ServiceDomains({ serviceId, serviceName }: { serviceId: string; serviceName: string }) {
  const { data: domains } = useDomains("services", serviceId);
  if (!domains?.length) return null;
  return (
    <div className="border border-outline-variant/60 rounded p-sm">
      <div className="flex items-center justify-between mb-sm">
        <Link
          to="/apps/$appId"
          params={{ appId: serviceId }}
          className="font-body-md text-body-md text-primary hover:text-primary-fixed-dim transition-colors"
        >
          {serviceName}
        </Link>
        <Badge tone="neutral">{domains.length} domains</Badge>
      </div>
      <table className="w-full text-left"><thead><tr className="text-label-caps text-muted-foreground"><th className="px-2 py-2 font-semibold">Host</th><th className="px-2 py-2 font-semibold">HTTPS</th><th className="px-2 py-2 font-semibold">Certificate</th></tr></thead><tbody>
        {domains.map((domain) => (
          <tr key={domain.id}>
            <td className="px-sm py-1.5 font-code-md text-code-md text-on-surface">{domain.host}</td>
            <td className="px-sm py-1.5"><Badge tone={domain.https ? "success" : "neutral"}>{domain.https ? "HTTPS" : "HTTP"}</Badge></td>
            <td className="px-sm py-1.5">
              {domain.https ? <Badge tone={domain.cert_status === "valid" || domain.cert_status === "issued" ? "success" : "warning"}>{domain.cert_status}</Badge> : <span className="font-code-md text-code-md text-on-surface-variant/50">—</span>}
            </td>
          </tr>
        ))}
      </tbody></table>
      <p className="font-body-sm text-body-sm text-on-surface-variant/60 mt-sm">{new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(new Date(domains[0].created_at))}</p>
    </div>
  );
}
