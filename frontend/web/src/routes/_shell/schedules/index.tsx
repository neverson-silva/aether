import { createFileRoute, Link } from "@tanstack/react-router";
import { CalendarBlank } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Card, EmptyState, Skeleton } from "@aether/design-system";
import { useAllCronJobs } from "../../../hooks";
import { PageHeader } from "../../../components/PageHeader";

function Schedules() {
  const query = useAllCronJobs();
  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-lg p-6 lg:p-8">
      <PageHeader
        eyebrow="Operations"
        title="Schedules"
        description="All cron jobs across your applications."
      />
      <Card variant="elevated" padding="none">
        <div className="overflow-hidden rounded-2xl border border-border">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[900px] text-left">
              <thead>
                <tr className="border-b border-border bg-surface-container-low/80 text-label-caps text-muted-foreground">
                  {[
                    "App",
                    "Job",
                    "Schedule",
                    "Command",
                    "Last run",
                    "Next run",
                    "Status",
                  ].map((header) => (
                    <th key={header} className="px-3 py-3">
                      {header}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {query.isLoading ? (
                  <tr>
                    <td colSpan={7} className="p-6">
                      <Skeleton variant="table" />
                    </td>
                  </tr>
                ) : (
                  (query.data ?? []).map((job) => (
                    <tr
                      key={job.id}
                      className="transition-[background-color] duration-150 hover:bg-surface-container"
                    >
                      <td className="px-3 py-3">
                        <Link
                          to="/apps/$appId"
                          params={{ appId: job.service_id ?? job.app_id }}
                          className="text-primary outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
                        >
                          {job.service_name}
                        </Link>
                      </td>
                      <td className="px-3 py-3 text-foreground">{job.name}</td>
                      <td className="px-3 py-3 font-mono text-code-md text-muted-foreground">
                        {job.schedule}
                      </td>
                      <td className="max-w-[320px] truncate px-3 py-3 font-mono text-code-md text-muted-foreground">
                        {job.command}
                      </td>
                      <td className="px-3 py-3 font-mono text-code-md text-muted-foreground">
                        {job.last_run || "—"}
                      </td>
                      <td className="px-3 py-3 font-mono text-code-md text-muted-foreground">
                        {job.next_run}
                      </td>
                      <td className="px-3 py-3">
                        <Badge tone={job.enabled ? "success" : "neutral"} dot>
                          {job.enabled ? "Active" : "Disabled"}
                        </Badge>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
        {!query.isLoading && !query.data?.length ? (
          <EmptyState
            icon={CalendarBlank as unknown as DesignIcon}
            title="No schedules yet"
            description="Create cron jobs inside an application."
          />
        ) : null}
      </Card>
    </main>
  );
}

export const Route = createFileRoute("/_shell/schedules/")({
  component: Schedules,
});
