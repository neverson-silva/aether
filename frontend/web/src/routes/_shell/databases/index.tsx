import { useState } from "react";
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
import { useBackupDatabase, useDatabases, useDeleteDatabase } from "../../../hooks";
import { getServer } from "../../../api/client";
import { DatabaseWizard } from "../../../components/DatabaseWizard";

type Database = NonNullable<ReturnType<typeof useDatabases>["data"]>[number];
const designIcon = (icon: typeof Plus) => icon as unknown as DesignIcon;

function statusFor(status: string) {
  if (["creating", "starting"].includes(status)) return { status: "deploying" as const, label: "Provisioning" };
  if (status === "ready") return { status: "healthy" as const, label: "Healthy" };
  if (status === "failed") return { status: "failed" as const, label: "Failed" };
  return { status: "unknown" as const, label: status };
}

function DatabasesPage() {
  const { data: databases, isLoading } = useDatabases();
  const deleteDatabase = useDeleteDatabase();
  const backupDatabase = useBackupDatabase();
  const { add } = useToast();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [dsn, setDsn] = useState<string | null>(null);

  const loadDsn = async () => {
    const database = databases?.[0];
    if (!database) return;
    const response = await fetch(`${getServer()}/api/v1/databases/${database.id}`, { credentials: "include" });
    const data = await response.json() as { dsn?: string };
    if (data.dsn) setDsn(data.dsn);
  };

  const columns = [
    { id: "name", header: "Name", accessor: (database: Database) => <Link to="/databases/$dbId" params={{ dbId: database.id }} className="font-medium text-foreground hover:text-primary">{database.name}</Link>, sortValue: (database: Database) => database.name },
    { id: "engine", header: "Engine", accessor: (database: Database) => <span className="font-mono text-body-sm text-muted-foreground">{database.engine}</span>, sortValue: (database: Database) => database.engine },
    { id: "version", header: "Version", accessor: (database: Database) => database.version || "Default" },
    { id: "status", header: "Status", accessor: (database: Database) => { const state = statusFor(database.status); return <RuntimeStatus status={state.status} label={state.label} live={state.status === "healthy" || state.status === "deploying"} />; } },
    { id: "actions", header: "", accessor: (database: Database) => (
      <div className="flex justify-end gap-2">
        <Button variant="ghost" size="sm" icon={designIcon(Archive)} loading={backupDatabase.isPending} onClick={() => backupDatabase.mutate(database.id, { onError: (error) => add({ title: "Backup failed", description: error.message, tone: "error" }), onSuccess: () => add({ title: "Backup started", tone: "success" }) })}>Backup</Button>
        <AlertDialog
          trigger={<Button variant="ghost" size="sm" icon={designIcon(Trash)} aria-label={`Delete ${database.name}`} />}
          title="Delete database"
          description={`The container and volume for ${database.name} will be removed. This action cannot be undone.`}
          confirmLabel="Delete database"
          onConfirm={() => deleteDatabase.mutate(database.id, { onError: (error) => add({ title: "Delete failed", description: error.message, tone: "error" }), onSuccess: () => add({ title: "Database deleted", tone: "success" }) })}
        />
      </div>
    ) },
  ];

  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-8 p-6 lg:p-8">
      <header className="flex flex-col gap-5 border-b border-border pb-6 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-2">
          <Typography as="p" level="label" tone="primary">Infrastructure</Typography>
          <Typography as="h1" level="display">Databases</Typography>
          <Typography as="p" level="body" tone="muted">Managed database workloads with credentials, backups and private networking.</Typography>
        </div>
        <Button icon={designIcon(Plus)} onClick={() => setWizardOpen(true)}>New database</Button>
      </header>

      {isLoading ? <Skeleton variant="table" className="h-56" aria-label="Loading databases" /> : null}
      {!isLoading && !databases?.length ? <EmptyState icon={designIcon(DatabaseIcon)} title="No databases" description="Provision PostgreSQL, MySQL, MariaDB, Redis or MongoDB with managed credentials." action={<Button icon={designIcon(Plus)} onClick={() => setWizardOpen(true)}>Create your first database</Button>} /> : null}
      {databases?.length ? <DataTable columns={columns} data={databases} rowId={(database) => database.id} empty="No databases found." /> : null}

      <Card as="section" variant="glass" padding="lg" header={<div className="flex items-center gap-3"><Eye size={20} className="text-primary" /><Typography as="h2" level="heading">Connection strings</Typography></div>}>
        <div className="space-y-4">
          <Typography as="p" level="body" tone="muted">Databases join the project network by hostname. Credentials are encrypted at rest and available only through authenticated requests.</Typography>
          {dsn ? <pre className="overflow-auto rounded-lg border border-border bg-surface-container p-4 font-mono text-body-sm text-foreground">{dsn}</pre> : <Button variant="outline" size="sm" onClick={loadDsn} disabled={!databases?.length}>Show DSN for first database</Button>}
        </div>
      </Card>

      <DatabaseWizard open={wizardOpen} onClose={() => setWizardOpen(false)} />
    </main>
  );
}

export const Route = createFileRoute("/_shell/databases/")({ component: DatabasesPage });
