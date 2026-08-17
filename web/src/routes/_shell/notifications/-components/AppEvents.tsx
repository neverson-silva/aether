import { StatusPill, fmtDate } from "../../../../components/ui";
import { useDeployments } from "../../../../hooks";

export function AppEvents({ app }: { app: { id: string; name: string } }) {
  const { data: deployments } = useDeployments(app.id);
  if (!deployments?.length) return null;
  return (
    <>
      {deployments.map((d) => (
        <div key={d.id} className="flex items-center gap-sm py-1 border-b border-outline-variant/40 last:border-0">
          <StatusPill status={d.status} />
          <span className="font-code-md text-code-md text-primary">{app.name}</span>
          <span className="font-code-md text-code-md text-on-surface-variant/60">#{d.number}</span>
          <span className="ml-auto font-code-md text-code-md text-on-surface-variant/50">{fmtDate(d.created_at)}</span>
        </div>
      ))}
    </>
  );
}
