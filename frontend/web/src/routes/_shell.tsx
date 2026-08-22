import { createFileRoute } from "@tanstack/react-router";
import { Shell } from "../components/shell";
import { requireAuth } from "../hooks/auth";

export const Route = createFileRoute("/_shell")({
  beforeLoad: async () => {
    const me = await requireAuth();
    return { me };
  },
  component: Shell,
});
