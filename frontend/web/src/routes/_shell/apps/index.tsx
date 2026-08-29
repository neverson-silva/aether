import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import {
  ArrowSquareOut,
  Cube,
  Database,
  HardDrives,
  Memory,
  Plus,
  RocketLaunch,
} from "@phosphor-icons/react";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  RuntimeStatus,
  Skeleton,
  Typography,
} from "@aether/design-system";
import type { Icon as DesignIcon } from "@aether/design-system";
import { useServices } from "../../../hooks";
import type { ServiceKind, ServiceStatus } from "../../../api/types";
import { CreateServiceLauncher } from "../../../components/CreateServiceLauncher";

const designIcon = (icon: typeof RocketLaunch) => icon as unknown as DesignIcon;

function displayStatus(status: ServiceStatus): "healthy" | "deploying" | "paused" | "failed" | "unknown" {
  if (status === "running") return "healthy";
  if (status === "deploying" || status === "pending") return "deploying";
  if (status === "stopped") return "paused";
  if (status === "failed") return "failed";
  return "unknown";
}

function ServiceCard({
  href,
  name,
  detail,
  port,
  memory,
  status,
  label,
  kind,
}: {
  href: string;
  name: string;
  detail: string;
  port: number;
  memory?: string;
  status: ServiceStatus;
  label?: string;
  kind: ServiceKind;
}) {
  const Icon = kind === "database" ? Database : Cube;
  return (
    <Link to={href} className="group block min-w-0" aria-label={`Open ${name}`}>
      <Card as="article" variant="interactive" padding="md" className="h-full">
        <div className="flex h-full min-h-40 flex-col gap-4">
          <div className="flex items-start justify-between gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Icon size={22} weight="duotone" aria-hidden="true" />
            </span>
            <RuntimeStatus status={displayStatus(status)} label={label} live={status === "running" || status === "deploying"} />
          </div>
          <div className="min-w-0 flex-1">
            <Typography as="h3" level="heading" truncate>{name}</Typography>
            <Typography as="p" level="code" tone="muted" truncate>{detail}</Typography>
          </div>
          <div className="flex items-center gap-3 border-t border-border pt-3 text-body-sm text-muted-foreground">
            <span className="inline-flex items-center gap-1.5">
              {port > 0 ? <><ArrowSquareOut size={15} aria-hidden="true" />:{port}</> : null}
            </span>
            {memory ? (
              <span className="inline-flex items-center gap-1.5">
                <Memory size={15} aria-hidden="true" />
                {memory}
              </span>
            ) : null}
            <ArrowSquareOut size={16} className="ml-auto transition-transform group-hover:translate-x-0.5" aria-hidden="true" />
          </div>
        </div>
      </Card>
    </Link>
  );
}

function Services() {
  const { data: services, isLoading } = useServices();
  const [launcherOpen, setLauncherOpen] = useState(false);
  const hasServices = Boolean(services?.length);

  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-8 p-6 lg:p-8">
      <header className="flex flex-col gap-5 border-b border-border pb-6 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-2">
          <Typography as="p" level="label" tone="primary">Runtime</Typography>
          <Typography as="h1" level="display">Services</Typography>
          <Typography as="p" level="body" tone="muted">Deploy, monitor and operate every application and database from one workspace.</Typography>
        </div>
        <Button icon={designIcon(Plus)} onClick={() => setLauncherOpen(true)}>New service</Button>
      </header>

      {isLoading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label="Loading services">
          {Array.from({ length: 4 }, (_, index) => <Skeleton key={index} variant="card" className="h-48" />)}
        </div>
      ) : null}

      {!isLoading && !hasServices ? (
        <EmptyState
          icon={designIcon(RocketLaunch)}
          title="No services yet"
          description="Deploy an application, provision a database or start from a ready-made template."
          action={<Button icon={designIcon(Plus)} onClick={() => setLauncherOpen(true)}>Create your first service</Button>}
        />
      ) : null}

      {!isLoading && hasServices ? (
        <section className="space-y-4" aria-labelledby="services-heading">
          <div className="flex items-center justify-between gap-4">
            <div>
              <Typography as="h2" id="services-heading" level="heading">Your services</Typography>
              <Typography as="p" level="small" tone="muted">Live resources in the selected workspace.</Typography>
            </div>
            <Badge tone="neutral" size="md" icon={designIcon(HardDrives)}>{services?.length ?? 0} services</Badge>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {(services ?? []).map((service) => (
              <ServiceCard
                key={service.id}
                href={`/apps/${service.id}`}
                name={service.name}
                detail={service.kind === "database" ? `${service.spec?.engine ?? "Database"} ${service.spec?.version ?? ""}` : service.kind === "compose" ? "Docker Compose stack" : service.spec?.source_type === "git" ? service.spec.git_url ?? "Git source" : service.spec?.image ?? "Application"}
                port={service.spec?.port ?? 0}
                memory={service.spec?.mem_mb ? `${service.spec.mem_mb >= 1024 ? service.spec.mem_mb / 1024 : service.spec.mem_mb} ${service.spec.mem_mb >= 1024 ? "GB" : "MB"}` : undefined}
                status={service.status}
                label={service.status === "pending" ? "Pending deployment" : undefined}
                kind={service.kind}
              />
            ))}
          </div>
        </section>
      ) : null}

      <CreateServiceLauncher open={launcherOpen} onClose={() => setLauncherOpen(false)} />
    </main>
  );
}

export const Route = createFileRoute("/_shell/apps/")({ component: Services });
