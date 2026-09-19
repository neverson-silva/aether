import { useEffect, useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Archive, Database as DatabaseIcon, Eye, Plus, Trash } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import {
  AlertDialog,
  Button,
  Card,
  DataTable,
  EmptyState,
  RuntimeStatus,
  Skeleton,
  Typography,
  useToast,
} from "@aether/design-system";
import { useServiceAction, useServiceConnection, useServices, useStartServiceDatabaseBackup } from "../../../hooks";
import type { ServiceSummary } from "../../../api/types";
import { DatabaseWizard } from "../../../components/DatabaseWizard";
import { PageHeader } from "../../../components/PageHeader";

const designIcon = (icon: typeof Plus) => icon as unknown as DesignIcon;

function statusFor(status: ServiceSummary["status"]) {
  if (status === "running") return { status: "healthy" as const, label: "Healthy" };
  if (status === "pending" || status === "deploying") return { status: "deploying" as const, label: "Provisioning" };
  if (status === "stopped") return { status: "paused" as const, label: "Stopped" };
  if (status === "failed") return { status: "failed" as const, label: "Failed" };
  return { status: "unknown" as const, label: "Unknown" };
}

function DatabasesPage() {
  const { data: services, isLoading } = useServices();
  const databases = (services ?? []).filter((service) => service.kind === "database");
  const deleteService = useServiceAction("delete");
  const backupDatabase = useStartServiceDatabaseBackup();
  const { add } = useToast();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [dsn, setDsn] = useState<string | null>(null);
  const [dsnOpen, setDsnOpen] = useState(false);
  const connection = useServiceConnection(databases[0]?.id ?? "", dsnOpen);

  const loadDsn = () => {
    if (databases[0]) setDsnOpen(true);
  };

  useEffect(() => {
    if (connection.data?.dsn) setDsn(connection.data.dsn);
  }, [connection.data?.dsn]);

  const columns = [
    { id: "name", header: "Name", accessor: (database: ServiceSummary) => <Link to="/apps/$appId" params={{ appId: database.id }} search={{ returnTo: "/databases" }} className="font-medium text-foreground hover:text-primary">{database.name}</Link>, sortValue: (database: ServiceSummary) => database.name },
    { id: "engine", header: "Engine", accessor: (database: ServiceSummary) => <span className="font-mono text-body-sm text-muted-foreground">{database.spec?.engine ?? "Database"}</span>, sortValue: (database: ServiceSummary) => database.spec?.engine ?? "" },
    { id: "version", header: "Version", accessor: (database: ServiceSummary) => database.spec?.version || "Default" },
    { id: "status", header: "Status", accessor: (database: ServiceSummary) => { const state = statusFor(database.status); return <RuntimeStatus status={state.status} label={state.label} live={state.status === "healthy" || state.status === "deploying"} />; } },
    { id: "actions", header: "", accessor: (database: ServiceSummary) => (
      <div className="flex justify-end gap-2">
        <Button variant="ghost" size="sm" icon={designIcon(Archive)} loading={backupDatabase.isPending} onClick={() => backupDatabase.mutate(database.id, { onError: (error) => add({ title: "Backup failed", description: error.message, tone: "error" }), onSuccess: () => add({ title: "Backup started", tone: "success" }) })}>Backup</Button>
        <AlertDialog
          trigger={<Button variant="danger" size="sm" icon={designIcon(Trash)} aria-label={`Delete ${database.name}`} />}
          title="Delete database"
          description={`The container and volume for ${database.name} will be removed. This action cannot be undone.`}
          confirmLabel="Delete database"
          onConfirm={() => deleteService.mutate(database.id, { onError: (error) => add({ title: "Delete failed", description: error.message, tone: "error" }), onSuccess: () => add({ title: "Database deleted", tone: "success" }) })}
        />
      </div>
    ) },
  ];

  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-8 p-6 lg:p-8">
      <PageHeader
        eyebrow="Infrastructure"
        title="Databases"
        description="Managed database workloads with credentials, backups and private networking."
        actions={<Button icon={designIcon(Plus)} onClick={() => setWizardOpen(true)}>New database</Button>}
      />

      {isLoading ? <Skeleton variant="table" className="h-56" aria-label="Loading databases" /> : null}
      {!isLoading && !databases?.length ? <EmptyState icon={designIcon(DatabaseIcon)} title="No databases" description="Provision PostgreSQL, MySQL, MariaDB, Redis or MongoDB with managed credentials." action={<Button icon={designIcon(Plus)} onClick={() => setWizardOpen(true)}>Create your first database</Button>} /> : null}
      {databases?.length ? <DataTable columns={columns} data={databases} rowId={(database) => database.id} empty="No databases found." /> : null}

      <Card as="section" variant="glass" padding="lg" header={<div className="flex items-center gap-3"><Eye size={20} className="text-primary" /><Typography as="h2" level="heading">Connection strings</Typography></div>}>
        <div className="space-y-4">
          <Typography as="p" level="body" tone="muted">Databases join the project network by hostname. Credentials are requested only when you explicitly reveal a connection string.</Typography>
          {dsn ? <pre className="overflow-auto rounded-lg border border-border bg-surface-container p-4 font-mono text-body-sm text-foreground">{dsn}</pre> : <Button variant="outline" size="sm" onClick={loadDsn} disabled={!databases?.length}>Show DSN for first database</Button>}
        </div>
      </Card>

      <DatabaseWizard open={wizardOpen} onClose={() => setWizardOpen(false)} />
    </main>
  );
}

export const Route = createFileRoute("/_shell/databases/")({ component: DatabasesPage });
