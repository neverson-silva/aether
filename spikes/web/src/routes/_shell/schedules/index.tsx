import { createFileRoute } from "@tanstack/react-router";
import { AppPage, AppPageHeader } from "../../../components/ds";
import { Link } from "@tanstack/react-router";
import { useAllCronJobs } from "../../../api/hooks";
import { StatusPill, Table } from "../../../components/ui";

function Schedules() {
  const { data: jobs, isLoading } = useAllCronJobs();
  return (
    <AppPage>
      <AppPageHeader
        title="Schedules"
        description="All cron jobs across your applications."
      />
      <div className="bg-surface-container-low border border-outline-variant rounded-lg">
        <Table headers={["App", "Job", "Schedule", "Command", "Last run", "Next run", "Status"]}>
          {isLoading && (
            <tr>
              <td colSpan={7} className="px-sm py-lg text-center font-body-sm text-body-sm text-on-surface-variant">Loading…</td>
            </tr>
          )}
          {(jobs ?? []).map((j) => (
            <tr key={j.id} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2">
                <Link to="/apps/$appId" params={{ appId: j.app_id }} className="font-body-md text-body-md text-primary hover:text-primary-fixed-dim transition-colors">
                  {j.app_name}
                </Link>
              </td>
              <td className="px-sm py-2 font-body-md text-body-md text-on-surface">{j.name}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{j.schedule}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant max-w-[320px] truncate">{j.command}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">{j.last_run || "—"}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{j.next_run}</td>
              <td className="px-sm py-2">
                <StatusPill status={j.enabled ? "active" : "disabled"} pulse={j.enabled} />
              </td>
            </tr>
          ))}
        </Table>
        {(jobs ?? []).length === 0 && !isLoading && (
          <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">
            No schedules yet. Create cron jobs inside an application.
          </p>
        )}
      </div>
    </AppPage>
  );
}

export const Route = createFileRoute("/_shell/schedules/")({
  component: Schedules,
});
