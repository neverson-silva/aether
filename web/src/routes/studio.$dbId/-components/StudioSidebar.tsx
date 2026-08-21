import { useState } from "react";
import { useRouter } from "@tanstack/react-router";
import {
  useDatabaseDetail,
  useStudioObjects,
  useStudioSchemas,
  useStudioTable,
  useStudioRenameTable,
  useStudioDropTable,
  useStudioAlterTable,
} from "../../../hooks";
import { Button, ConfirmDialog, Input, Modal, useToast } from "../../../components/ui";
import { ContextMenu, type ContextMenuState } from "./ContextMenu";
import { CreateTableModal } from "./CreateTableModal";

interface Sel {
  schema: string;
  table: string;
}

const TYPE_ICON: Record<string, string> = {
  table: "table",
  view: "visibility",
  "materialized view": "view_column",
  collection: "view_list",
  key: "key",
  function: "code",
  procedure: "functions",
};

function typeColor(t: string): string {
  switch (t) {
    case "table":
      return "text-on-surface-variant";
    case "view":
    case "materialized view":
      return "text-[#ecc48d]";
    case "function":
    case "procedure":
      return "text-[#82aaff]";
    case "collection":
      return "text-[#f78c6c]";
    case "key":
      return "text-[#89ddff]";
    default:
      return "text-on-surface-variant";
  }
}

export function StudioSidebar({
  dbId,
  selected,
  onSelect,
  schemas,
  onSchemaChanged,
}: {
  dbId: string;
  selected: Sel | null;
  onSelect: (sel: Sel) => void;
  schemas: string[];
  onSchemaChanged: () => Promise<void>;
}) {
  const router = useRouter();
  const { data } = useDatabaseDetail(dbId);
  const schemasQ = useStudioSchemas(dbId);
  const { toast } = useToast();
  const [expandedSchemas, setExpandedSchemas] = useState<Record<string, boolean>>({});
  const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>({});
  const [menu, setMenu] = useState<ContextMenuState | null>(null);
  const [editTarget, setEditTarget] = useState<Sel | null>(null);
  const [renameTarget, setRenameTarget] = useState<Sel | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Sel | null>(null);
  const [renaming, setRenaming] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [newName, setNewName] = useState("");
  const [savingEdit, setSavingEdit] = useState(false);

  const rename = useStudioRenameTable(dbId);
  const drop = useStudioDropTable(dbId);
  const alter = useStudioAlterTable(dbId);

  const db = data?.database;
  const running = db?.status === "running" || db?.status === "ready";

  const toggleSchema = (s: string) => setExpandedSchemas((p) => ({ ...p, [s]: !p[s] }));
  const toggleGroup = (k: string) => setExpandedGroups((p) => ({ ...p, [k]: !p[k] }));

  const schemaList = schemasQ.data ?? schemas;

  const refresh = async (msg = "Schema refreshed") => {
    try {
      await onSchemaChanged();
      toast(msg);
    } catch (err) {
      toast(err instanceof Error ? err.message : "refresh failed", "error");
    }
  };

  const openTableMenu = (e: React.MouseEvent, sel: Sel) => {
    e.preventDefault();
    e.stopPropagation();
    onSelect(sel);
    setMenu({
      x: e.clientX,
      y: e.clientY,
      items: [
        { label: "Edit Table", icon: "edit", onSelect: () => setEditTarget(sel) },
        { label: "Rename Table", icon: "drive_file_rename_outline", onSelect: () => { setRenameTarget(sel); setNewName(sel.table); } },
        { label: "Delete Table", icon: "delete", danger: true, onSelect: () => setDeleteTarget(sel) },
      ],
    });
  };

  const openSchemaMenu = (e: React.MouseEvent, schema: string) => {
    e.preventDefault();
    e.stopPropagation();
    setMenu({
      x: e.clientX,
      y: e.clientY,
      items: [{ label: "Refresh", icon: "refresh", onSelect: () => void refresh() }],
    });
  };

  const confirmRename = async () => {
    if (!renameTarget || !newName.trim() || newName === renameTarget.table) return;
    setRenaming(true);
    try {
      await rename.mutateAsync({ schema: renameTarget.schema, table: renameTarget.table, name: newName.trim() });
      setRenameTarget(null);
      toast("Table renamed");
      await onSchemaChanged();
      onSelect({ schema: renameTarget.schema, table: newName.trim() });
    } catch (err) {
      toast(err instanceof Error ? err.message : "rename failed", "error");
    } finally {
      setRenaming(false);
    }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await drop.mutateAsync({ schema: deleteTarget.schema, table: deleteTarget.table });
      setDeleteTarget(null);
      toast("Table deleted");
      if (selected?.schema === deleteTarget.schema && selected?.table === deleteTarget.table) {
        onSelect(deleteTarget.schema === selected.schema ? { schema: selected.schema, table: "" } : selected);
      }
      await onSchemaChanged();
    } catch (err) {
      toast(err instanceof Error ? err.message : "delete failed", "error");
    } finally {
      setDeleting(false);
    }
  };

  const saveEdit = async (payload: { table: string; schema: string; columns: { name: string; type: string; nullable: boolean; primary: boolean; default?: string }[] }) => {
    setSavingEdit(true);
    try {
      await alter.mutateAsync({ schema: payload.schema, table: payload.table, columns: payload.columns });
      toast("Table updated");
      await onSchemaChanged();
    } catch (err) {
      toast(err instanceof Error ? err.message : "update failed", "error");
    } finally {
      setSavingEdit(false);
    }
  };

  return (
    <aside className="w-60 flex-shrink-0 bg-surface-container-lowest border-r border-outline-variant h-full flex flex-col overflow-hidden">
      <div className="px-md py-lg border-b border-outline-variant">
        <div className="flex items-center gap-sm mb-1">
          <span className="material-symbols-outlined text-[16px] text-primary">dns</span>
          <span className="font-code-md text-[12px] font-bold text-on-surface truncate">{db?.name ?? "Database"}</span>
        </div>
        <div className="flex items-center gap-sm">
          <span className="material-symbols-outlined text-[12px] text-on-surface-variant">public</span>
          <span className="font-label-caps text-[10px] text-on-surface-variant">{db?.engine}</span>
          <span className="w-1 h-1 rounded-full bg-outline-variant" />
          <span className="w-1.5 h-1.5 rounded-full bg-[#82aaff]" />
          <span className="font-label-caps text-[10px] text-on-surface-variant capitalize">{running ? "online" : "offline"}</span>
        </div>
      </div>

      <div className="px-sm py-sm border-b border-outline-variant">
        <button
          onClick={() => router.history.back()}
          className="flex items-center gap-sm py-sm px-sm text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-all rounded-md font-label-caps text-label-caps cursor-pointer"
        >
          <span className="material-symbols-outlined text-[18px]">arrow_back</span>
          Back
        </button>
      </div>

      <div className="flex-1 overflow-y-auto sidebar-scroll py-md pr-xs">
        <div className="font-label-caps text-label-caps text-on-surface-variant mb-sm px-sm flex items-center justify-between">
          Object Explorer
          <button onClick={() => void refresh()} className="text-on-surface-variant hover:text-primary transition-colors" title="Refresh">
            <span className="material-symbols-outlined text-[16px]">refresh</span>
          </button>
        </div>

        {schemasQ.isLoading && (
          <div className="px-sm py-md font-body-sm text-body-sm text-on-surface-variant/60">
            Connecting to database...
          </div>
        )}

        {schemasQ.isError && (
          <div className="mx-sm my-md p-md rounded-lg border border-error/40 bg-error/10">
            <div className="flex items-center gap-sm font-label-caps text-label-caps text-error mb-1">
              <span className="material-symbols-outlined text-[14px]">error</span>
              Object Explorer unavailable
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-md break-words">
              {(schemasQ.error as Error)?.message || "Could not connect to the database."}
            </p>
            <button onClick={() => schemasQ.refetch()} className="font-label-caps text-label-caps text-primary hover:underline">
              Retry
            </button>
          </div>
        )}

        {!schemasQ.isLoading && !schemasQ.isError && (
          <div className="flex flex-col text-[12px] font-code-md">
            <div className="flex items-center gap-xs py-1 px-sm text-on-surface" onContextMenu={(e) => openSchemaMenu(e, "")}>
              <span className="material-symbols-outlined text-[14px] text-on-surface-variant">expand_more</span>
              <span className="material-symbols-outlined text-[14px] text-[#89ddff]">database</span>
              <span className="truncate">{db?.name ?? "Database"}</span>
            </div>

            {schemaList.map((schema) => {
              const open = expandedSchemas[schema];
              return (
                <div key={schema} className="ml-4 relative">
                  <div
                    className="flex items-center gap-xs py-1 px-sm hover:bg-surface-container-high rounded cursor-pointer group"
                    onClick={() => toggleSchema(schema)}
                    onContextMenu={(e) => openSchemaMenu(e, schema)}
                  >
                    <span className="material-symbols-outlined text-[14px] text-on-surface-variant group-hover:text-primary transition-colors">
                      {open ? "expand_more" : "chevron_right"}
                    </span>
                    <span className="material-symbols-outlined text-[14px] text-[#c792ea]">schema</span>
                    <span className="truncate">{schema}</span>
                  </div>

                  {open && (
                    <SchemaGroup
                      schema={schema}
                      dbId={dbId}
                      expanded={expandedGroups}
                      onToggle={toggleGroup}
                      selected={selected}
                      onSelect={onSelect}
                      onTableContext={openTableMenu}
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="mt-auto pt-md border-t border-outline-variant px-sm">
        <a className="flex items-center gap-sm py-sm px-sm text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-all rounded-md font-label-caps text-label-caps cursor-pointer" href="#">
          <span className="material-symbols-outlined text-[16px]">description</span>
          Documentation
        </a>
        <a className="flex items-center gap-sm py-sm px-sm text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-all rounded-md font-label-caps text-label-caps cursor-pointer" href="#">
          <span className="material-symbols-outlined text-[16px]">support_agent</span>
          Support
        </a>
      </div>

      <ContextMenu menu={menu} onClose={() => setMenu(null)} />

      <EditTableModal open={!!editTarget} onClose={() => setEditTarget(null)} dbId={dbId} target={editTarget} schemas={schemaList} engine={db?.engine} saving={savingEdit} onSave={saveEdit} />

      <Modal open={!!renameTarget} onClose={() => setRenameTarget(null)} title="Rename Table" size="sm">
        <div className="space-y-lg">
          <Field label="Current name">
            <div className="px-3 py-2 bg-surface-container border border-outline-variant rounded-lg font-code-md text-code-md text-on-surface">{renameTarget?.table}</div>
          </Field>
          <Field label="New name">
            <Input icon="drive_file_rename_outline" value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="new_table_name" />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setRenameTarget(null)}>Cancel</Button>
            <Button type="button" loading={renaming} disabled={!newName.trim() || newName === renameTarget?.table} onClick={() => void confirmRename()}>
              Rename
            </Button>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void confirmDelete()}
        title="Delete Table"
        description={`Are you sure you want to delete "${deleteTarget?.schema}.${deleteTarget?.table}"? This action cannot be undone.`}
        confirmLabel="Delete Table"
        danger
        requireType={deleteTarget?.table}
      />
    </aside>
  );
}

function EditTableModal({
  open,
  onClose,
  dbId,
  target,
  schemas,
  engine,
  saving,
  onSave,
}: {
  open: boolean;
  onClose: () => void;
  dbId: string;
  target: Sel | null;
  schemas: string[];
  engine: string | undefined;
  saving: boolean;
  onSave: (payload: { table: string; schema: string; columns: { name: string; type: string; nullable: boolean; primary: boolean; default?: string }[] }) => void;
}) {
  const { data: detail } = useStudioTable(dbId, target?.schema ?? "", target?.table ?? "");
  if (!target) return null;
  const columns = (detail?.columns ?? []).map((c) => ({
    name: c.name,
    type: c.type,
    nullable: c.nullable ?? true,
    primary: c.primary_key ?? false,
    default: c.default ?? "",
  }));
  return (
    <CreateTableModal
      open={open}
      onClose={onClose}
      schemas={schemas}
      onCreate={() => {}}
      engine={engine}
      edit={target ? { schema: target.schema, table: target.table, columns } : null}
      onSave={onSave}
      saving={saving}
    />
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-2">
      <span className="font-label-caps text-label-caps text-on-surface-variant">{label}</span>
      {children}
    </label>
  );
}

function SchemaGroup({
  schema,
  dbId,
  expanded,
  onToggle,
  selected,
  onSelect,
  onTableContext,
}: {
  schema: string;
  dbId: string;
  expanded: Record<string, boolean>;
  onToggle: (k: string) => void;
  selected: Sel | null;
  onSelect: (sel: Sel) => void;
  onTableContext: (e: React.MouseEvent, sel: Sel) => void;
}) {
  const objectsQ = useStudioObjects(dbId, schema);
  const objects = objectsQ.data ?? [];
  const tables = (objects ?? []).filter((o) => o.type === "table");
  const views = (objects ?? []).filter((o) => o.type === "view" || o.type === "materialized view");
  const funcs = (objects ?? []).filter((o) => o.type === "function" || o.type === "procedure");
  const others = (objects ?? []).filter((o) => !["table", "view", "materialized view", "function", "procedure"].includes(o.type));

  const tableKey = `${schema}:tables`;
  const viewKey = `${schema}:views`;
  const funcKey = `${schema}:functions`;
  const otherKey = `${schema}:others`;

  if (objectsQ.isLoading) {
    return <div className="ml-8 py-1 px-sm font-body-sm text-body-sm text-on-surface-variant/50">Loading objects...</div>;
  }

  if (objectsQ.isError) {
    return (
      <div className="ml-8 my-1 px-sm py-2 rounded border border-error/30 bg-error/5">
        <p className="font-body-sm text-body-sm text-on-surface-variant mb-sm break-words">
          {(objectsQ.error as Error)?.message || "Failed to load objects."}
        </p>
        <button onClick={() => objectsQ.refetch()} className="font-label-caps text-label-caps text-primary hover:underline">
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="ml-4">
      <GroupRow
        label={`Tables (${tables.length})`}
        icon="table_chart"
        color="text-[#f78c6c]"
        open={!!expanded[tableKey]}
        onToggle={() => onToggle(tableKey)}
      >
        {tables.map((t) => {
          const active = selected?.schema === schema && selected?.table === t.name;
          return (
            <button
              key={t.name}
              onClick={() => onSelect({ schema, table: t.name })}
              onContextMenu={(e) => onTableContext(e, { schema, table: t.name })}
              className={`flex items-center gap-xs py-1 px-sm rounded cursor-pointer w-full text-left group ${active ? "bg-surface-container-high" : "hover:bg-surface-container-high"}`}
            >
              <span className="material-symbols-outlined text-[14px] opacity-0">chevron_right</span>
              <span className={`material-symbols-outlined text-[14px] ${active ? "text-primary" : "text-on-surface-variant group-hover:text-primary"}`}>table</span>
              <span className={`truncate ${active ? "text-primary" : ""}`}>{t.name}</span>
            </button>
          );
        })}
      </GroupRow>

      <GroupRow label={`Views (${views.length})`} icon="visibility" color="text-[#ecc48d]" open={!!expanded[viewKey]} onToggle={() => onToggle(viewKey)}>
        {views.map((v) => (
          <button
            key={v.name}
            onClick={() => onSelect({ schema, table: v.name })}
            className="flex items-center gap-xs py-1 px-sm hover:bg-surface-container-high rounded cursor-pointer w-full text-left group"
          >
            <span className="material-symbols-outlined text-[14px] opacity-0">chevron_right</span>
            <span className="material-symbols-outlined text-[14px] text-[#ecc48d] group-hover:text-primary">visibility</span>
            <span className="truncate">{v.name}</span>
          </button>
        ))}
      </GroupRow>

      <GroupRow label={`Functions (${funcs.length})`} icon="code" color="text-[#82aaff]" open={!!expanded[funcKey]} onToggle={() => onToggle(funcKey)}>
        {funcs.map((f) => (
          <div key={f.name} className="flex items-center gap-xs py-1 px-sm text-on-surface-variant">
            <span className="material-symbols-outlined text-[14px] opacity-0">chevron_right</span>
            <span className="material-symbols-outlined text-[14px] text-[#82aaff]">code</span>
            <span className="truncate">{f.name}</span>
          </div>
        ))}
      </GroupRow>

      {others.length > 0 && (
        <GroupRow label={`Other (${others.length})`} icon="category" color="text-on-surface-variant" open={!!expanded[otherKey]} onToggle={() => onToggle(otherKey)}>
          {others.map((o) => (
            <div key={o.name} className="flex items-center gap-xs py-1 px-sm text-on-surface-variant">
              <span className="material-symbols-outlined text-[14px] opacity-0">chevron_right</span>
              <span className={`material-symbols-outlined text-[14px] ${typeColor(o.type)}`}>{TYPE_ICON[o.type] ?? "category"}</span>
              <span className="truncate">{o.name}</span>
            </div>
          ))}
        </GroupRow>
      )}
    </div>
  );
}

function GroupRow({
  label,
  icon,
  color,
  open,
  onToggle,
  children,
}: {
  label: string;
  icon: string;
  color: string;
  open: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}) {
  return (
    <div>
      <button onClick={onToggle} className="flex items-center gap-xs py-1 px-sm hover:bg-surface-container-high rounded cursor-pointer w-full text-left group">
        <span className="material-symbols-outlined text-[14px] text-on-surface-variant group-hover:text-primary transition-colors">
          {open ? "expand_more" : "chevron_right"}
        </span>
        <span className={`material-symbols-outlined text-[14px] ${color}`}>{icon}</span>
        <span className="truncate text-on-surface-variant">{label}</span>
      </button>
      {open && <div className="ml-2">{children}</div>}
    </div>
  );
}