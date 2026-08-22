import { Badge } from "@aether/design-system";
import { useDeployments } from "../../../../hooks";

function formatDate(iso: string) { return new Intl.DateTimeFormat("en", { dateStyle: "medium", timeStyle: "short" }).format(new Date(iso)); }

export function AppEvents({ app }: { app: { id: string; name: string } }) {
  const { data: deployments } = useDeployments(app.id);
  if (!deployments?.length) return null;
  return (
    <>
      {deployments.map((d) => (
        <div key={d.id} className="flex items-center gap-sm py-1 border-b border-outline-variant/40 last:border-0">
          <Badge tone={d.status === "ready" || d.status === "success" ? "success" : d.status === "failed" ? "danger" : "neutral"}>{d.status}</Badge>
          <span className="font-code-md text-code-md text-primary">{app.name}</span>
          <span className="font-code-md text-code-md text-on-surface-variant/60">#{d.number}</span>
          <span className="ml-auto font-code-md text-code-md text-on-surface-variant/50">{formatDate(d.created_at)}</span>
        </div>
      ))}
    </>
  );
}
