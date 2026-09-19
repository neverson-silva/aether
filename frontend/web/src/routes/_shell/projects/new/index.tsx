import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_shell/projects/new/")({
  beforeLoad: () => {
    throw redirect({ to: "/projects", search: { create: true } });
  },
  component: () => null,
});
