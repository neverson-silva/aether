import { createFileRoute } from "@tanstack/react-router";
import { useEffect } from "react";
import { z } from "zod";
import { EmptyState, Skeleton } from "@aether/design-system";
import { useDatabaseDetail } from "../../../hooks";

function DatabaseRouteAlias() {
  const { dbId } = Route.useParams();
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const { data, isLoading, error } = useDatabaseDetail(dbId);
  const serviceId = data?.database.service_id;

  useEffect(() => {
    if (!serviceId) return;
    void navigate({
      to: "/apps/$appId",
      params: { appId: serviceId },
      search: { returnTo: search.returnTo ?? "/databases" },
      replace: true,
    });
  }, [navigate, search.returnTo, serviceId]);

  if (isLoading) return <Skeleton variant="card" className="min-h-32" />;
  if (error || !serviceId) {
    return (
      <EmptyState
        title="Database service is unavailable"
        description="This database does not have a canonical service identity yet. Refresh after the platform migration completes."
      />
    );
  }
  return <Skeleton variant="card" className="min-h-32" />;
}

export const Route = createFileRoute("/_shell/databases/$dbId/")({
  validateSearch: z.object({ returnTo: z.string().optional() }),
  component: DatabaseRouteAlias,
});
