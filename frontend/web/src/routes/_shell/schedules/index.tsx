import { createFileRoute, Link } from "@tanstack/react-router";
import { CalendarBlank } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Card, EmptyState, Skeleton } from "@aether/design-system";
import { useAllCronJobs } from "../../../hooks";

function Schedules() {
  const query = useAllCronJobs();
  return <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-8 p-6 lg:p-8"><header><p className="text-label-caps text-primary">Operations</p><h1 className="text-headline-sm font-semibold text-foreground">Schedules</h1><p className="mt-1 text-body-md text-muted-foreground">All cron jobs across your applications.</p></header><Card padding="none"><div className="overflow-x-auto"><table className="w-full min-w-[900px] text-left"><thead><tr className="border-b border-border text-label-caps text-muted-foreground">{["App", "Job", "Schedule", "Command", "Last run", "Next run", "Status"].map((header) => <th key={header} className="px-3 py-3">{header}</th>)}</tr></thead><tbody className="divide-y divide-border">{query.isLoading ? <tr><td colSpan={7} className="p-6"><Skeleton variant="table" /></td></tr> : (query.data ?? []).map((job) => <tr key={job.id} className="hover:bg-surface-container"><td className="px-3 py-3"><Link to="/apps/$appId" params={{ appId: job.service_id ?? job.app_id }} className="text-primary hover:underline">{job.service_name}</Link></td><td className="px-3 py-3 text-foreground">{job.name}</td><td className="px-3 py-3 font-mono text-code-md text-muted-foreground">{job.schedule}</td><td className="max-w-[320px] truncate px-3 py-3 font-mono text-code-md text-muted-foreground">{job.command}</td><td className="px-3 py-3 font-mono text-code-md text-muted-foreground">{job.last_run || "—"}</td><td className="px-3 py-3 font-mono text-code-md text-muted-foreground">{job.next_run}</td><td className="px-3 py-3"><Badge tone={job.enabled ? "success" : "neutral"} dot>{job.enabled ? "Active" : "Disabled"}</Badge></td></tr>)}</tbody></table></div>{!query.isLoading && !query.data?.length ? <EmptyState icon={CalendarBlank as unknown as DesignIcon} title="No schedules yet" description="Create cron jobs inside an application." /> : null}</Card></main>;
}

export const Route = createFileRoute("/_shell/schedules/")({ component: Schedules });
