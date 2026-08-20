import { AppPage, AppPageHeader } from "../../../components/ds";
import { useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import {
  useBackupDatabase,
  useDatabases,
  useDeleteDatabase,
} from "../../../hooks";
import { getServer } from "../../../api/client";
import { DatabaseWizard } from "../../../components/DatabaseWizard";
import {
  Button,
  Card,
  ConfirmDialog,
  EmptyState,
  StatusPill,
  Table,
  useToast,
} from "../../../components/ui";

function DatabasesPage() {
  const { data: databases, isLoading } = useDatabases();
  const deleteDb = useDeleteDatabase();
  const backupDb = useBackupDatabase();
  const { toast } = useToast();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const [dsn, setDsn] = useState<string | null>(null);

  return (
    <AppPage>
      <AppPageHeader
        title="Databases"
        description="Managed databases as OCI workloads with integrated credentials and backups."
        actions={
          <Button leftIcon="add" onClick={() => setWizardOpen(true)}>
            New database
          </Button>
        }
      />

      {isLoading && <div className="py-md" />}
      {!isLoading && !databases?.length && (
        <EmptyState
          icon="storage"
          title="No databases"
          description="Provision PostgreSQL, MySQL, MariaDB, Redis or MongoDB with managed credentials."
          action={<Button onClick={() => setWizardOpen(true)}>Create your first database</Button>}
        />
      )}
      {!!databases?.length && (
        <div className="bg-surface-container-low border border-outline-variant rounded-lg">
          <Table headers={["Name", "Engine", "Version", "Status", ""]}>
            {databases.map((db) => (
              <tr key={db.id} className="hover:bg-surface-container-high transition-colors">
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface">{db.name}</td>
                <td className="px-sm py-2 font-body-sm text-body-sm text-on-surface-variant">{db.engine}</td>
                <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">{db.version || "default"}</td>
                <td className="px-sm py-2">
                  <StatusPill
                    status={["creating", "starting"].includes(db.status) ? "pending deploy" : db.status}
                    pulse={["creating", "starting"].includes(db.status)}
                  />
                </td>
                <td className="px-sm py-2">
                  <div className="flex items-center gap-sm justify-end">
                    <button
                      onClick={() =>
                        backupDb.mutate(db.id, {
                          onError: (e) => toast(e.message, "error"),
                        })
                      }
                      className="flex items-center gap-1 font-label-caps text-label-caps text-tertiary hover:text-tertiary-fixed-dim transition-colors uppercase"
                    >
                      <span className="material-symbols-outlined text-[14px]">backup</span>
                      Backup
                    </button>
                    <button
                      onClick={() => setConfirmId(db.id)}
                      className="material-symbols-outlined text-[16px] text-on-surface-variant hover:text-error transition-colors"
                    >
                      delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </Table>
        </div>
      )}

      <Card>
        <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
          Connection strings
        </h2>
        <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
          Databases join the project network as <code className="font-code-md text-code-md text-primary">aether-db-&lt;name&gt;</code> —
          apps on the same network reach them by hostname. Credentials are encrypted at rest.
        </p>
        {dsn && (
          <pre className="bg-surface-container-lowest border border-outline-variant rounded-lg p-md font-code-md text-code-md text-on-surface overflow-auto">
            {dsn}
          </pre>
        )}
        {!dsn && (
          <button
            onClick={() => {
              const db = databases?.[0];
              if (!db) return;
              apiGet(`/api/v1/databases/${db.id}`).then((r: { dsn: string }) => setDsn(r.dsn));
            }}
            className="text-primary font-body-sm text-body-sm hover:text-primary-fixed-dim transition-colors"
          >
            Show DSN for first database
          </button>
        )}
      </Card>

      <DatabaseWizard open={wizardOpen} onClose={() => setWizardOpen(false)} />

      <ConfirmDialog
        open={!!confirmId}
        onClose={() => setConfirmId(null)}
        onConfirm={() =>
          deleteDb.mutate(confirmId!, {
            onError: (e) => toast(e.message, "error"),
          })
        }
        title="Delete database"
        description="The container and volume will be removed. This cannot be undone."
        confirmLabel="Delete"
        danger
      />
    </AppPage>
  );
}

function apiGet(path: string): Promise<{ dsn: string }> {
  const server = getServer();
  return fetch(server + path, { credentials: "include" }).then((r) => r.json());
}

export const Route = createFileRoute("/_shell/databases/")({
  component: DatabasesPage,
});
