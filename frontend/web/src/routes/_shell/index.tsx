import {
  ArrowsDownUp,
  Export,
  Globe,
  RocketLaunch,
  ShieldCheck,
} from "@phosphor-icons/react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import {
  Button,
  Card,
  MetricCard,
  Progress,
  RuntimeStatus,
  useToast,
} from "@aether/design-system";
import { useExportOrg, useSystemSummary } from "../../hooks";

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  if (bytes < 1024 * 1024 * 1024)
    return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GiB`;
}

function Dashboard() {
  const navigate = useNavigate();
  const { data: summary } = useSystemSummary();
  const exportOrg = useExportOrg();
  const toast = useToast();
  const health = summary?.health_pct ?? 0;
  const traffic = formatBytes(summary?.traffic_bytes ?? 0);
  const deployments = summary?.deployments ?? 0;
  const projects = summary?.projects ?? [];
  const [projectPage, setProjectPage] = useState(0);
  const projectsPerPage = 4;
  const projectPageCount = Math.max(
    1,
    Math.ceil(projects.length / projectsPerPage),
  );
  const visibleProjects = useMemo(
    () =>
      projects.slice(
        projectPage * projectsPerPage,
        (projectPage + 1) * projectsPerPage,
      ),
    [projectPage, projects],
  );

  const exportReport = async () => {
    try {
      const yaml = await exportOrg.mutateAsync();
      const url = URL.createObjectURL(
        new Blob([yaml], { type: "application/yaml" }),
      );
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = "aether.yml";
      anchor.click();
      URL.revokeObjectURL(url);
      toast.add({ title: "Exported aether.yml", tone: "success" });
    } catch (error) {
      toast.add({
        title: error instanceof Error ? error.message : "Export failed",
        tone: "error",
      });
    }
  };

  return (
    <div className="space-y-8">
      <header className="flex flex-col justify-between gap-4 md:flex-row md:items-end">
        <div>
          <p className="text-label-caps uppercase tracking-[0.18em] text-muted-foreground">
            Platform overview
          </p>
          <h1 className="mt-2 text-display-lg font-semibold text-foreground">
            Fleet Overview
          </h1>
          <div className="mt-3 flex items-center gap-2 text-body-sm text-muted-foreground">
            <RuntimeStatus status="healthy" label="System operational" live />
            <span>Updated just now</span>
          </div>
        </div>
        <Button
          variant="outline"
          loading={exportOrg.isPending}
          onClick={exportReport}
        >
          <Export size={18} />
          Export report
        </Button>
      </header>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <MetricCard
          label="Global health"
          value={health.toFixed(1)}
          unit="%"
          delta="From live probes"
          trend="up"
        />
        <MetricCard
          label="Total traffic"
          value={traffic}
          unit="net"
          delta="Since container start"
          trend="flat"
        />
        <MetricCard
          label="Deployments"
          value={String(deployments)}
          unit="runs"
          delta="Across all services"
          trend="flat"
        />
      </section>

      <section className="grid grid-cols-1 items-stretch gap-4 xl:grid-cols-[minmax(0,1.6fr)_minmax(20rem,1fr)]">
        <Card variant="elevated" padding="none">
          <div className="flex items-center justify-between border-b border-border px-5 py-4">
            <div>
              <h2 className="text-headline-sm font-semibold text-foreground">
                Active projects
              </h2>
              <p className="mt-1 text-body-sm text-muted-foreground">
                Current environments and deployment state.
              </p>
            </div>
            <Link
              to="/projects"
              className="text-body-sm font-semibold text-primary hover:underline"
            >
              View all
            </Link>
          </div>
          <div className="flex min-h-56 flex-col">
            <div className="flex-1 divide-y divide-border overflow-hidden">
              {projects.length === 0 ? (
                <div className="p-8 text-center text-body-sm text-muted-foreground">
                  No projects yet. Create one to get started.
                </div>
              ) : (
                visibleProjects.map((project) => (
                  <button
                    key={project.id}
                    type="button"
                    onClick={() =>
                      navigate({
                        to: "/projects/$projectId",
                        params: { projectId: project.id },
                      })
                    }
                    className="flex w-full items-center gap-4 px-5 py-4 text-left transition-colors hover:bg-surface-container"
                  >
                    <span className="flex size-10 shrink-0 items-center justify-center rounded-lg border border-border bg-surface-container text-primary">
                      <Globe size={20} weight="duotone" />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-semibold text-foreground">
                        {project.name}
                      </span>
                      <span className="mt-1 block text-body-sm text-muted-foreground">
                        {project.env}
                      </span>
                    </span>
                    <RuntimeStatus
                      status={
                        project.status === "healthy"
                          ? "healthy"
                          : project.status === "syncing"
                            ? "deploying"
                            : project.status === "degraded"
                              ? "degraded"
                              : "unknown"
                      }
                      label={project.status}
                    />
                    <span className="hidden text-code-md text-muted-foreground md:block">
                      {project.last_deploy}
                    </span>
                  </button>
                ))
              )}
            </div>
            {projectPageCount > 1 ? (
              <div className="flex items-center justify-between border-t border-border px-5 py-3 text-body-sm text-muted-foreground">
                <span>
                  Page {projectPage + 1} of {projectPageCount}
                </span>
                <div className="flex gap-2">
                  <button
                    type="button"
                    disabled={projectPage === 0}
                    onClick={() =>
                      setProjectPage((page) => Math.max(0, page - 1))
                    }
                    className="rounded-md px-2 py-1 hover:bg-surface-container disabled:pointer-events-none disabled:opacity-40"
                  >
                    Previous
                  </button>
                  <button
                    type="button"
                    disabled={projectPage === projectPageCount - 1}
                    onClick={() =>
                      setProjectPage((page) =>
                        Math.min(projectPageCount - 1, page + 1),
                      )
                    }
                    className="rounded-md px-2 py-1 hover:bg-surface-container disabled:pointer-events-none disabled:opacity-40"
                  >
                    Next
                  </button>
                </div>
              </div>
            ) : null}
          </div>
        </Card>

        <Card
          variant="elevated"
          header={
            <div>
              <h2 className="text-headline-sm font-semibold text-foreground">
                Resource usage
              </h2>
              <p className="mt-1 text-body-sm text-muted-foreground">
                Host allocation at a glance.
              </p>
            </div>
          }
        >
          <div className="space-y-6">
            <ResourceMeter
              label="CPU allocation"
              value={summary?.cpu_pct ?? 0}
              icon={ShieldCheck}
            />
            <ResourceMeter
              label="Memory"
              value={summary?.mem_pct ?? 0}
              icon={ArrowsDownUp}
              status="success"
            />
            <ResourceMeter
              label="Storage I/O"
              value={summary?.io_pct ?? 0}
              icon={RocketLaunch}
              status="warning"
            />
          </div>
        </Card>
      </section>
    </div>
  );
}

function ResourceMeter({
  label,
  value,
  icon: Icon,
  status = "default",
}: {
  label: string;
  value: number;
  icon: typeof ShieldCheck;
  status?: "default" | "success" | "warning";
}) {
  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <span className="flex items-center gap-2 text-label-caps uppercase text-muted-foreground">
          <Icon size={16} />
          {label}
        </span>
        <span className="font-mono text-code-md text-foreground">
          {Math.round(value)}%
        </span>
      </div>
      <Progress
        value={value}
        status={status}
        label={`${label}: ${Math.round(value)} percent`}
      />
    </div>
  );
}

export const Route = createFileRoute("/_shell/")({ component: Dashboard });
