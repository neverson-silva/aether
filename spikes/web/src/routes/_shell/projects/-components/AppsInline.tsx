import { Link } from "@tanstack/react-router";
import { StatusPill, EmptyState, Table } from "../../../../components/ui";
import { AppLink } from "../../../../components/ds";
import { useApps } from "../../../../api/hooks";

export function AppsInline() {
  const { data: apps, isLoading } = useApps();
  if (isLoading) return <div className="py-md" />;
  return (
    <div className="mt-lg">
      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
        Applications
      </h2>
      {!apps?.length && (
        <EmptyState
          icon="rocket_launch"
          title="No applications"
          description="Create an application from an OCI image or a git repository."
          action={
            <AppLink to="/apps" variant="primary" size="sm">
              Create application
            </AppLink>
          }
        />
      )}
      {!!apps?.length && (
        <div className="bg-surface-container-low border border-outline-variant rounded-lg">
          <Table headers={["Nome", "Source", "Image / Repository", "Port"]}>
            {apps.map((app) => (
              <tr key={app.id} className="hover:bg-surface-container-high transition-colors">
                <td className="px-sm py-2">
                  <Link
                    to="/apps/$appId"
                    params={{ appId: app.id }}
                    className="font-body-md text-body-md text-primary hover:text-primary-fixed-dim transition-colors"
                  >
                    {app.name}
                  </Link>
                </td>
                <td className="px-sm py-2">
                  <StatusPill status={app.source_type} />
                </td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">
                  {app.source_type === "image" ? app.image : app.git_url}
                </td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">:{app.port}</td>
              </tr>
            ))}
          </Table>
        </div>
      )}
    </div>
  );
}
