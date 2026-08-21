import { createFileRoute } from "@tanstack/react-router";
import { requireAuth } from "../../hooks/auth";

export const Route = createFileRoute("/studio/$dbId/")({
  beforeLoad: async () => {
    const me = await requireAuth();
    return { me };
  },
});
