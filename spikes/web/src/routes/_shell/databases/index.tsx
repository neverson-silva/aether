import { AppPage, AppPageHeader } from "../../../components/ds";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { createFileRoute } from "@tanstack/react-router";
import {
  useBackupDatabase,
  useCreateDatabase,
  useDatabases,
  useDeleteDatabase,
  useProjects,
} from "../../../api/hooks";
import {
  Button,
  Card,
  ConfirmDialog,
  EmptyState,
  Field,
  Input,
  Modal,
  Select,
  StatusPill,
  Table,
  useToast,
} from "../../../components/ui";

const DB_CATEGORIES = [
  {
    name: "Relational (SQL)",
    engines: [
      { value: "postgres", label: "PostgreSQL", icon: "🐘" },
      { value: "mysql", label: "MySQL", icon: "🐬" },
      { value: "mariadb", label: "MariaDB", icon: "🦭" },
      { value: "mssql", label: "SQL Server", icon: "🔷" },
      { value: "oracle", label: "Oracle", icon: "🔶" },
    ],
  },
  {
    name: "Document (NoSQL)",
    engines: [{ value: "mongodb", label: "MongoDB", icon: "🍃" }],
  },
  {
    name: "In-memory / Cache",
    engines: [{ value: "redis", label: "Redis", icon: "🔴" }],
  },
];

const schema = z.object({
  project_id: z.string().min(1, "Project is required"),
  name: z.string().min(1, "Name is required").regex(/^[a-z0-9-_]+$/, "lowercase only"),
  engine: z.enum(["postgres", "mysql", "mariadb", "redis", "mongodb", "mssql", "oracle"]),
  version: z.string().optional(),
});

function DatabasesPage() {
  const { data: databases, isLoading } = useDatabases();
  const { data: projects } = useProjects();
  const createDb = useCreateDatabase();
  const deleteDb = useDeleteDatabase();
  const backupDb = useBackupDatabase();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const [dsn, setDsn] = useState<string | null>(null);

  const {
    register,
    setValue,
    watch,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { engine: "postgres", version: "" },
  });
  const engineValue = watch("engine");

  const submit = async (values: z.infer<typeof schema>) => {
    try {
      await createDb.mutateAsync({
        project_id: values.project_id,
        name: values.name,
        engine: values.engine,
        version: values.version || undefined,
      });
      toast("Database provisioning started");
      setOpen(false);
      reset();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create database", "error");
    }
  };

  return (
    <AppPage>
      <AppPageHeader
        title="Databases"
        description="Managed databases as OCI workloads with integrated credentials and backups."
        actions={
          <Button leftIcon="add" onClick={() => setOpen(true)}>
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
          action={<Button onClick={() => setOpen(true)}>Create your first database</Button>}
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
                    status={["creating", "starting"].includes(db.status) ? "provisioning" : db.status}
                    pulse={["creating", "starting"].includes(db.status)}
                  />
                </td>
                <td className="px-sm py-2">
                  <div className="flex items-center gap-sm justify-end">
                    <button
                      onClick={() =>
                        backupDb.mutate(db.id, {
                          onSuccess: () => toast("Backup created"),
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

      <Modal open={open} onClose={() => setOpen(false)} title="New database">
        <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
            <Field label="Project" hint={errors.project_id?.message}>
              <Select {...register("project_id")}>
                <option value="">Select...</option>
                {(projects ?? []).map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Name" hint={errors.name?.message}>
              <Input icon="storage" placeholder="main-db" {...register("name")} />
            </Field>
            <Field label="Engine" hint={errors.engine?.message}>
              <div className="space-y-md">
                {DB_CATEGORIES.map((cat) => (
                  <div key={cat.name}>
                    <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase mb-sm">{cat.name}</p>
                    <div className="grid grid-cols-2 gap-sm">
                      {cat.engines.map((eng) => (
                        <button
                          key={eng.value}
                          type="button"
                          onClick={() => setValue("engine", eng.value as never, { shouldValidate: true })}
                          className={`flex items-center gap-sm px-sm py-2 rounded border transition-colors text-left ${
                            engineValue === eng.value ? "border-primary bg-primary/10" : "border-outline-variant hover:border-primary/40 bg-surface-container-low"
                          }`}
                        >
                          <span className="text-[18px]">{eng.icon}</span>
                          <span className="font-body-sm text-body-sm text-on-surface">{eng.label}</span>
                        </button>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </Field>
            <Field label="Version" hint={errors.version?.message || "postgres: 16/15/14 · mysql: 8/8.0 · mariadb: 11/10.11 · redis: 7/6 · mongo: 7/6 · mssql: 2022 · oracle: 23 — empty = default"}>
              <Input icon="tag" placeholder="16" {...register("version")} />
            </Field>
          </div>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Provisioning..." : "Create database"}
            </Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={!!confirmId}
        onClose={() => setConfirmId(null)}
        onConfirm={() =>
          deleteDb.mutate(confirmId!, {
            onSuccess: () => toast("Database removed"),
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
  const server = localStorage.getItem("aether_server") || "";
  return fetch(server + path, { credentials: "include" }).then((r) => r.json());
}

export const Route = createFileRoute("/_shell/databases/")({
  component: DatabasesPage,
});
