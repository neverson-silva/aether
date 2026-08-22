import { useState } from "react";
import { useRouter } from "@tanstack/react-router";
import { ArrowLeft, ArrowsClockwise, CaretDown, CaretRight, ChatCircle, Code, Database, FileText, Globe, Key, List, Package, PencilSimple, Question, Table, Trash, Wrench, X, Warning } from "@phosphor-icons/react";
import {
  useDatabaseDetail,
  useStudioObjects,
  useStudioSchemas,
  useStudioTable,
  useStudioRenameTable,
  useStudioDropTable,
  useStudioAlterTable,
} from "../../../hooks";
import { AlertDialog, Button, Dialog, Field, Input, Skeleton, useToast } from "@aether/design-system";
import { ContextMenu, type ContextMenuState } from "./ContextMenu";
import { CreateTableModal } from "./CreateTableModal";

interface Sel {
  schema: string;
  table: string;
}

const TYPE_ICON: Record<string, typeof Table> = {
  table: Table,
  view: Globe,
  "materialized view": List,
  collection: Package,
  key: Key,
  function: Code,
  procedure: Wrench,
};

const IconFor = ({ name, className = "", size = 16 }: { name: string; className?: string; size?: number }) => {
  const IconComponent = ({ dns: Database, public: Globe, arrow_back: ArrowLeft, refresh: ArrowsClockwise, error: Warning, expand_more: CaretDown, chevron_right: CaretRight, database: Database, schema: List, description: FileText, support_agent: Question, edit: PencilSimple, drive_file_rename_outline: PencilSimple, delete: Trash, code: Code, table: Table, table_chart: Table, visibility: Globe, key: Key, data_object: Code, functions: Wrench, category: Package, chat: ChatCircle } as Record<string, typeof Table>)[name] ?? Package;
  return <IconComponent size={size} className={className} aria-hidden="true" />;
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
  const { add } = useToast();
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
      add({ title: msg, tone: "success" });
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Refresh failed", tone: "error" });
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
      add({ title: "Table renamed", tone: "success" });
      await onSchemaChanged();
      onSelect({ schema: renameTarget.schema, table: newName.trim() });
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Rename failed", tone: "error" });
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
      add({ title: "Table deleted", tone: "success" });
      if (selected?.schema === deleteTarget.schema && selected?.table === deleteTarget.table) {
        onSelect(deleteTarget.schema === selected.schema ? { schema: selected.schema, table: "" } : selected);
      }
      await onSchemaChanged();
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Delete failed", tone: "error" });
    } finally {
      setDeleting(false);
    }
  };

  const saveEdit = async (payload: { table: string; schema: string; columns: { name: string; type: string; nullable: boolean; primary: boolean; default?: string }[] }) => {
    setSavingEdit(true);
    try {
      await alter.mutateAsync({ schema: payload.schema, table: payload.table, columns: payload.columns });
      add({ title: "Table updated", tone: "success" });
      await onSchemaChanged();
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Update failed", tone: "error" });
    } finally {
      setSavingEdit(false);
    }
  };

  return (
    <aside className="w-60 flex-shrink-0 bg-surface-container-lowest border-r border-outline-variant h-full flex flex-col overflow-hidden">
      <div className="px-md py-lg border-b border-outline-variant">
        <div className="flex items-center gap-sm mb-1">
          <IconFor name="dns" className="text-primary" />
          <span className="font-code-md text-[12px] font-bold text-on-surface truncate">{db?.name ?? "Database"}</span>
        </div>
        <div className="flex items-center gap-sm">
          <IconFor name="public" className="text-on-surface-variant" size={12} />
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
          <IconFor name="arrow_back" size={18} />
          Back
        </button>
      </div>

      <div className="flex-1 overflow-y-auto sidebar-scroll py-md pr-xs">
        <div className="font-label-caps text-label-caps text-on-surface-variant mb-sm px-sm flex items-center justify-between">
          Object Explorer
          <button onClick={() => void refresh()} className="text-on-surface-variant hover:text-primary transition-colors" title="Refresh">
            <IconFor name="refresh" />
          </button>
        </div>

        {schemasQ.isLoading && (
          <div className="px-sm py-md font-body-sm text-body-sm text-on-surface-variant/60">
            <div className="space-y-sm"><Skeleton variant="text" /><Skeleton variant="text" className="w-2/3" /></div>
          </div>
        )}

        {schemasQ.isError && (
          <div className="mx-sm my-md p-md rounded-lg border border-error/40 bg-error/10">
            <div className="flex items-center gap-sm font-label-caps text-label-caps text-error mb-1">
              <IconFor name="error" size={14} />
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
              <IconFor name="expand_more" size={14} className="text-on-surface-variant" />
              <IconFor name="database" size={14} className="text-[#89ddff]" />
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
                    <IconFor name={open ? "expand_more" : "chevron_right"} size={14} className="text-on-surface-variant group-hover:text-primary transition-colors" />
                    <IconFor name="schema" size={14} className="text-[#c792ea]" />
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
          <IconFor name="description" />
          Documentation
        </a>
        <a className="flex items-center gap-sm py-sm px-sm text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-all rounded-md font-label-caps text-label-caps cursor-pointer" href="#">
          <IconFor name="support_agent" />
          Support
        </a>
      </div>

      <ContextMenu menu={menu} onClose={() => setMenu(null)} />

      <EditTableModal open={!!editTarget} onClose={() => setEditTarget(null)} dbId={dbId} target={editTarget} schemas={schemaList} engine={db?.engine} saving={savingEdit} onSave={saveEdit} />

      <Dialog trigger={<span />} open={!!renameTarget} onOpenChange={(value) => { if (!value) setRenameTarget(null); }} title="Rename Table">
        <div className="space-y-lg">
          <Field label="Current name">
            <div className="px-3 py-2 bg-surface-container border border-outline-variant rounded-lg font-code-md text-code-md text-on-surface">{renameTarget?.table}</div>
          </Field>
          <Field label="New name">
            <Input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="new_table_name" />
          </Field>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button type="button" variant="ghost" onClick={() => setRenameTarget(null)}>Cancel</Button>
            <Button type="button" loading={renaming} disabled={!newName.trim() || newName === renameTarget?.table} onClick={() => void confirmRename()}>
              Rename
            </Button>
          </div>
        </div>
      </Dialog>

      <AlertDialog
        trigger={<span />}
        open={!!deleteTarget}
        onOpenChange={(value) => { if (!value) setDeleteTarget(null); }}
        onConfirm={() => void confirmDelete()}
        title="Delete Table"
        description={`Are you sure you want to delete "${deleteTarget?.schema}.${deleteTarget?.table}"? This action cannot be undone.`}
        confirmLabel="Delete Table"
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

function ObjectRow({
  dbId,
  schema,
  object,
  selected,
  onSelect,
  onContextMenu,
}: {
  dbId: string;
  schema: string;
  object: { name: string; type: string };
  selected: Sel | null;
  onSelect: (sel: Sel) => void;
  onContextMenu?: (e: React.MouseEvent, sel: Sel) => void;
}) {
  const [open, setOpen] = useState(false);
  const detailQ = useStudioTable(dbId, schema, object.name, open);
  const active = selected?.schema === schema && selected?.table === object.name;
  const columns = detailQ.data?.columns ?? [];

  return (
    <div>
      <div
        className={`flex items-center gap-xs py-1 px-sm rounded cursor-pointer w-full text-left group ${active ? "bg-surface-container-high" : "hover:bg-surface-container-high"}`}
        onContextMenu={onContextMenu ? (e) => onContextMenu(e, { schema, table: object.name }) : undefined}
      >
        <button
          type="button"
          onClick={() => setOpen((value) => !value)}
          className="flex items-center justify-center w-4 h-4 text-on-surface-variant hover:text-primary"
          aria-label={`${open ? "Collapse" : "Expand"} ${object.type} ${object.name}`}
        >
          <IconFor name={open ? "expand_more" : "chevron_right"} size={14} />
        </button>
        <button
          type="button"
          onClick={() => onSelect({ schema, table: object.name })}
          className="flex items-center gap-xs min-w-0 flex-1 text-left"
        >
          {(() => { const IconComponent = TYPE_ICON[object.type] ?? Table; return <IconComponent size={14} className={active ? "text-primary" : typeColor(object.type)} aria-hidden="true" />; })()}
          <span className={`truncate ${active ? "text-primary" : ""}`}>{object.name}</span>
        </button>
      </div>
      {open && (
        <div className="ml-8 border-l border-outline-variant pl-sm">
          {detailQ.isLoading && <div className="space-y-xs py-1"><Skeleton variant="text" /><Skeleton variant="text" className="w-2/3" /></div>}
          {detailQ.isError && <div className="py-1 font-body-sm text-body-sm text-error">Could not load columns.</div>}
          {!detailQ.isLoading && !detailQ.isError && columns.length === 0 && (
            <div className="py-1 font-body-sm text-body-sm text-on-surface-variant/50">No columns</div>
          )}
          {!detailQ.isLoading && !detailQ.isError && columns.map((column) => (
            <div key={column.name} className="flex items-center gap-xs py-1 text-on-surface-variant" title={`${column.name}: ${column.type}`}>
              <IconFor name={column.primary_key ? "key" : "data_object"} size={13} className="text-[#89ddff]" />
              <span className="truncate">{column.name}</span>
              <span className="ml-auto truncate text-[10px] text-on-surface-variant/60">{column.type}</span>
            </div>
          ))}
        </div>
      )}
    </div>
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
    return <div className="ml-8 space-y-xs py-1 px-sm"><Skeleton variant="text" /><Skeleton variant="text" className="w-2/3" /></div>;
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
        {tables.map((t) => (
          <ObjectRow
            key={t.name}
            dbId={dbId}
            schema={schema}
            object={t}
            selected={selected}
            onSelect={onSelect}
            onContextMenu={onTableContext}
          />
        ))}
      </GroupRow>

      <GroupRow label={`Views (${views.length})`} icon="visibility" color="text-[#ecc48d]" open={!!expanded[viewKey]} onToggle={() => onToggle(viewKey)}>
        {views.map((v) => (
          <ObjectRow
            key={v.name}
            dbId={dbId}
            schema={schema}
            object={v}
            selected={selected}
            onSelect={onSelect}
          />
        ))}
      </GroupRow>

      <GroupRow label={`Functions (${funcs.length})`} icon="code" color="text-[#82aaff]" open={!!expanded[funcKey]} onToggle={() => onToggle(funcKey)}>
        {funcs.map((f) => (
          <div key={f.name} className="flex items-center gap-xs py-1 px-sm text-on-surface-variant">
            <CaretRight size={14} className="opacity-0" aria-hidden="true" />
            <IconFor name="code" size={14} className="text-[#82aaff]" />
            <span className="truncate">{f.name}</span>
          </div>
        ))}
      </GroupRow>

      {others.length > 0 && (
        <GroupRow label={`Other (${others.length})`} icon="category" color="text-on-surface-variant" open={!!expanded[otherKey]} onToggle={() => onToggle(otherKey)}>
          {others.map((o) => (
            <div key={o.name} className="flex items-center gap-xs py-1 px-sm text-on-surface-variant">
              <CaretRight size={14} className="opacity-0" aria-hidden="true" />
              {(() => { const IconComponent = TYPE_ICON[o.type] ?? Package; return <IconComponent size={14} className={typeColor(o.type)} aria-hidden="true" />; })()}
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
        <IconFor name={open ? "expand_more" : "chevron_right"} size={14} className="text-on-surface-variant group-hover:text-primary transition-colors" />
        <IconFor name={icon} size={14} className={color} />
        <span className="truncate text-on-surface-variant">{label}</span>
      </button>
      {open && <div className="ml-2">{children}</div>}
    </div>
  );
}
