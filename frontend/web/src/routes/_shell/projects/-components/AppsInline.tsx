import { Link } from "@tanstack/react-router";
import { RocketLaunch } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, EmptyState, DataTable, Button, Skeleton } from "@aether/design-system";
import { useApps } from "../../../../hooks";

export function AppsInline() {
  const { data: apps, isLoading } = useApps();
  if (isLoading) return <div className="mt-lg space-y-sm" aria-label="Loading applications"><Skeleton variant="text" className="w-40" /><Skeleton variant="table" /><Skeleton variant="table" /></div>;
  return (
    <div className="mt-lg">
      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
        Applications
      </h2>
      {!apps?.length && (
        <EmptyState
            icon={RocketLaunch as unknown as DesignIcon}
          title="No applications"
          description="Create an application from an OCI image or a git repository."
          action={
            <Link to="/apps"><Button variant="primary"><RocketLaunch size={16} aria-hidden="true" />Create application</Button></Link>
          }
        />
      )}
      {!!apps?.length && (
        <div className="bg-surface-container-low border border-outline-variant rounded-lg">
          <DataTable
            columns={[
              { id: "name", header: "Name", accessor: (app) => <Link to="/apps/$appId" params={{ appId: app.id }} className="text-primary hover:underline">{app.name}</Link> },
              { id: "source", header: "Source", accessor: (app) => <Badge tone="neutral">{app.source_type}</Badge> },
              { id: "image", header: "Image / Repository", accessor: (app) => app.source_type === "image" ? app.image : app.git_url },
              { id: "port", header: "Port", accessor: (app) => `:${app.port}` },
            ]}
            data={apps}
            rowId={(app) => app.id}
          />
        </div>
      )}
    </div>
  );
}
