import { createFileRoute, redirect } from "@tanstack/react-router";
import { Shell } from "../components/shell";
import { apiGet } from "../api/client";

export const Route = createFileRoute("/_shell")({
  beforeLoad: async () => {
    try {
      await apiGet("/api/v1/me");
    } catch {
      throw redirect({ to: "/login" });
    }
  },
  component: Shell,
});
