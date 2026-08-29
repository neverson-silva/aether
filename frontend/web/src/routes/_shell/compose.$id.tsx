import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { z } from "zod";

export const Route = createFileRoute("/_shell/compose/$id")({
  validateSearch: z.object({ returnTo: z.string().optional() }),
  component: ComposeRouteAlias,
});

function ComposeRouteAlias() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const search = Route.useSearch();

  useEffect(() => {
    void navigate({
      to: "/apps/$appId",
      params: { appId: id },
      search: { kind: "compose", returnTo: search.returnTo },
      replace: true,
    });
  }, [id, navigate, search.returnTo]);

  return null;
}
