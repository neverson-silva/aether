import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { z } from "zod";
import { useComposeStack } from "../../hooks/use-compose-stack";

export const Route = createFileRoute("/_shell/compose/$id")({
  validateSearch: z.object({ returnTo: z.string().optional() }),
  component: ComposeRouteAlias,
});

function ComposeRouteAlias() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const search = Route.useSearch();
  const compose = useComposeStack(id);

  useEffect(() => {
    if (compose.isPending) return;
    void navigate({
      to: "/apps/$appId",
      params: { appId: compose.data?.service_id ?? id },
      search: { returnTo: search.returnTo },
      replace: true,
    });
  }, [compose.data?.service_id, compose.isPending, id, navigate, search.returnTo]);

  return null;
}
