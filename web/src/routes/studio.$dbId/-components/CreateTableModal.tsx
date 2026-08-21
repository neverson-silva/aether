import { useEffect, useMemo, useRef, useState } from "react";
import { Button, Field, Input, Modal } from "../../../components/ui";
import { cn } from "../../../components/ui";

interface ColumnDef {
  name: string;
  type: string;
  nullable: boolean;
  primary: boolean;
  default: string;
}

const DEFAULT_COLUMN: ColumnDef = { name: "", type: "TEXT", nullable: true, primary: false, default: "" };

const TYPE_OPTIONS = [
  "TEXT", "VARCHAR", "INTEGER", "BIGINT", "SERIAL", "BIGSERIAL", "UUID", "BOOLEAN",
  "TIMESTAMP", "TIMESTAMPTZ", "DATE", "TIME", "DECIMAL", "NUMERIC", "JSON", "JSONB",
  "BYTEA", "ARRAY", "ENUM", "GEOMETRY", "INET", "CIDR", "MONEY", "INTERVAL",
];

function quoteIdent(engine: string | undefined, name: string): string {
  if (engine === "mysql" || engine === "mariadb") return `\`${name.replace(/`/g, "``")}\``;
  if (engine === "mssql") return `[${name}]`;
  return `"${name.replace(/"/g, '""')}"`;
}

function buildCreateTableSQL(engine: string | undefined, schema: string, table: string, columns: ColumnDef[]): string {
  const lines: string[] = [];
  const primaryCols: string[] = [];
  const errors: string[] = [];
  const seen = new Set<string>();
  for (const c of columns) {
    const name = c.name.trim();
    if (!name) continue;
    if (seen.has(name.toLowerCase())) {
      errors.push(`Duplicate column name "${name}"`);
      continue;
    }
    seen.add(name.toLowerCase());
    const quoted = quoteIdent(engine, name);
    let line = `  ${quoted} ${c.type}`;
    if (c.primary) {
      line += " PRIMARY KEY";
      primaryCols.push(quoted);
    } else {
      if (!c.nullable) line += " NOT NULL";
      if (c.default !== "") line += ` DEFAULT ${c.default}`;
    }
    lines.push(line);
  }
  if (lines.length === 0) {
    errors.push("Add at least one column with a name to generate the statement.");
    return "";
  }
  const schemaQ = quoteIdent(engine, schema);
  const tableQ = quoteIdent(engine, table);
  const qualified = engine === "mysql" || engine === "mariadb" ? tableQ : `${schemaQ}.${tableQ}`;
  let stmt = `CREATE TABLE ${qualified} (\n${lines.join(",\n")}`;
  if (primaryCols.length > 1) {
    stmt += `,\n  PRIMARY KEY (${primaryCols.join(", ")})`;
  }
  stmt += "\n);";
  return stmt;
}

export function CreateTableModal({
  open,
  onClose,
  engine,
  schemas,
  onCreate,
  edit,
  onSave,
  saving,
}: {
  open: boolean;
  onClose: () => void;
  engine: string | undefined;
  schemas: string[];
  onCreate: (payload: { sql: string; table: string; schema: string; columns: ColumnDef[] }) => void;
  edit?: { schema: string; table: string; columns: ColumnDef[] } | null;
  onSave?: (payload: { table: string; schema: string; columns: ColumnDef[] }) => void;
  saving?: boolean;
}) {
  const [table, setTable] = useState("");
  const [schema, setSchema] = useState("");
  const [columns, setColumns] = useState<ColumnDef[]>([]);
  const [showSql, setShowSql] = useState(false);
  const nameRefs = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => {
    if (open && edit) {
      setTable(edit.table);
      setSchema(edit.schema);
      setColumns(edit.columns.length ? edit.columns : []);
    } else if (open) {
      setTable("");
      setSchema(schemas[0] ?? "public");
      setColumns([]);
    }
  }, [open, edit, schemas]);

  const sql = useMemo(() => buildCreateTableSQL(engine, schema, table, columns), [engine, schema, table, columns]);

  const updateColumn = (idx: number, patch: Partial<ColumnDef>) => {
    setColumns((cols) => cols.map((c, i) => (i === idx ? { ...c, ...patch } : c)));
  };

  const togglePrimary = (idx: number) => {
    setColumns((cols) => cols.map((c, i) => (i === idx ? { ...c, primary: !c.primary } : c)));
  };

  const addColumn = () => {
    const n = columns.length + 1;
    setColumns((cols) => [...cols, { ...DEFAULT_COLUMN, name: `column_${n}`, default: "" }]);
    requestAnimationFrame(() => nameRefs.current[columns.length]?.focus());
  };

  const removeColumn = (idx: number) => {
    setColumns((cols) => cols.filter((_, i) => i !== idx));
  };

  const moveColumn = (idx: number, dir: -1 | 1) => {
    setColumns((cols) => {
      const next = [...cols];
      const target = idx + dir;
      if (target < 0 || target >= next.length) return cols;
      [next[idx], next[target]] = [next[target], next[idx]];
      return next;
    });
  };

  const onKeyDown = (e: React.KeyboardEvent, idx: number) => {
    if (e.key === "Enter") {
      e.preventDefault();
      addColumn();
    } else if (e.key === "Delete" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      removeColumn(idx);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      moveColumn(idx, -1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      moveColumn(idx, -1);
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      moveColumn(idx, 1);
    }
  };

  const columnCount = columns.length;
  const nameError = table.trim() === "" ? "Table name is required" : !/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(table.trim()) ? "Use letters, digits and underscores, starting with a letter" : "";

  return (
    <Modal open={open} onClose={onClose} title={edit ? "Edit Table" : "Create Table"} wide>
      <div className="flex flex-col gap-lg">
        <div className="flex flex-wrap items-end gap-lg">
          {edit ? (
            <div className="flex flex-wrap items-end gap-lg">
              <div className="flex-1 min-w-56">
                <Field label="Table Name">
                  <div className="px-3 py-2 bg-surface-container border border-outline-variant rounded-lg font-code-md text-code-md text-on-surface">{edit.table}</div>
                </Field>
              </div>
              <div className="w-56">
                <Field label="Schema">
                  <div className="px-3 py-2 bg-surface-container border border-outline-variant rounded-lg font-code-md text-code-md text-on-surface">{edit.schema}</div>
                </Field>
              </div>
            </div>
          ) : (
            <div className="flex flex-wrap items-end gap-lg">
              <div className="flex-1 min-w-56">
                <Field label="Table Name" hint={nameError || undefined}>
                  <Input icon="table" placeholder="e.g. user_profiles" value={table} onChange={(e) => setTable(e.target.value)} aria-invalid={!!nameError} />
                </Field>
              </div>
              <div className="w-56">
                <Field label="Schema">
                  <select
                    value={schema}
                    onChange={(e) => setSchema(e.target.value)}
                    className="w-full bg-surface-container border border-outline-variant rounded-lg px-md py-[10px] font-body-md text-body-md text-on-surface"
                  >
                    {schemas.map((s) => (
                      <option key={s} value={s}>{s}</option>
                    ))}
                  </select>
                </Field>
              </div>
            </div>
          )}
        </div>

        <div>
          <div className="flex items-center justify-between mb-2">
            <div className="font-label-caps text-label-caps text-on-surface-variant uppercase tracking-wider">Columns</div>
            <button
              onClick={addColumn}
              className="text-body-sm text-primary hover:text-primary-container flex items-center gap-1 transition-colors"
            >
              <span className="material-symbols-outlined text-[16px]">add</span>
              Add column
            </button>
          </div>

          <div className="overflow-y-auto max-h-[45vh] sidebar-scroll border border-outline-variant rounded-lg">
            <div className="grid grid-cols-12 gap-x-3 gap-y-px px-3 py-2 bg-surface-container-lowest border-b border-outline-variant text-label-caps text-label-caps text-on-surface-variant sticky top-0 z-10">
              <div className="col-span-1 flex justify-center">PK</div>
              <div className="col-span-3">Name</div>
              <div className="col-span-3">Type</div>
              <div className="col-span-2 flex justify-center">Nullable</div>
              <div className="col-span-3">Default</div>
            </div>

            {columns.map((c, idx) => {
              const nameErr = c.name.trim() === "" ? "" : undefined;
              return (
                <div
                  key={idx}
                  onKeyDown={(e) => onKeyDown(e, idx)}
                  className="grid grid-cols-12 gap-x-3 gap-y-px px-3 py-1 items-center bg-surface-container-low border-b border-outline-variant/50 last:border-b-0 hover:bg-surface-container-high/40 transition-colors group"
                >
                  <div className="col-span-1 flex justify-center">
                    <button
                      onClick={() => togglePrimary(idx)}
                      className={cn(
                        "w-4 h-4 rounded-full border-2 flex items-center justify-center transition-colors",
                        c.primary ? "border-primary bg-primary/20" : "border-outline-variant hover:border-on-surface-variant"
                      )}
                      aria-label={c.primary ? "Primary key" : "Not primary key"}
                    >
                      {c.primary && <span className="w-2 h-2 rounded-full bg-primary" />}
                    </button>
                  </div>
                  <div className="col-span-3">
                    <input
                      ref={(el) => { nameRefs.current[idx] = el; }}
                      className="bg-transparent border border-transparent rounded px-1 py-1 text-code-md text-code-md text-on-surface w-full focus:border-primary focus:bg-background outline-none transition-colors"
                      type="text"
                      value={c.name}
                      placeholder="column_name"
                      onChange={(e) => updateColumn(idx, { name: e.target.value })}
                      aria-invalid={!!nameErr}
                    />
                  </div>
                  <div className="col-span-3">
                    <select
                      value={c.type}
                      onChange={(e) => updateColumn(idx, { type: e.target.value })}
                      className="w-full bg-surface-container rounded border border-outline-variant px-2 py-1 text-code-md text-code-md text-secondary"
                    >
                      {TYPE_OPTIONS.map((t) => (
                        <option key={t} value={t}>{t}</option>
                      ))}
                    </select>
                  </div>
                  <div className="col-span-2 flex justify-center">
                    <button
                      onClick={() => updateColumn(idx, { nullable: !c.nullable })}
                      className={cn(
                        "flex items-center gap-1 px-1.5 py-0.5 rounded border font-label-caps text-label-caps transition-colors",
                        c.nullable ? "border-outline-variant text-on-surface-variant" : "border-primary/40 text-primary"
                      )}
                      aria-label={c.nullable ? "Nullable: on" : "Nullable: off"}
                    >
                      {c.nullable ? "ON" : "OFF"}
                    </button>
                  </div>
                  <div className="col-span-3">
                    <input
                      className="bg-transparent border border-outline-variant/30 rounded px-1.5 py-1 text-code-md text-code-md text-on-surface w-full focus:border-primary focus:bg-background outline-none transition-colors"
                      type="text"
                      value={c.default}
                      placeholder="NULL"
                      onChange={(e) => updateColumn(idx, { default: e.target.value })}
                    />
                  </div>
                </div>
              );
            })}

            {columns.length === 0 && (
              <div className="py-6 text-center font-body-sm text-body-sm text-on-surface-variant/60">
                No columns yet — add your first column to define the table schema.
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center justify-between gap-md">
          {!edit && (
          <button
            onClick={() => setShowSql((s) => !s)}
            className={cn(
              "text-body-sm font-body-sm flex items-center gap-2 px-3 py-1.5 rounded border transition-colors",
              showSql ? "border-primary/40 text-primary bg-primary/10" : "border-outline-variant text-on-surface-variant hover:text-on-surface"
            )}
          >
            <span className="material-symbols-outlined text-[16px]">code</span>
            Preview SQL
            <span className={cn("material-symbols-outlined text-[14px] transition-transform", showSql && "rotate-180")}>expand_more</span>
          </button>
          )}
          {edit && <div />}
          {nameError && <span className="text-error text-body-sm font-body-sm">{nameError}</span>}
          <div className="flex items-center gap-md ml-auto">
            <Button type="button" variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="button"
              disabled={!sql || !!nameError}
              onClick={() => {
                onCreate({ sql, table: table.trim(), schema, columns });
                onClose();
              }}
            >
              Create table
            </Button>
          </div>
        </div>

        {edit && (
          <p className="font-body-sm text-body-sm text-on-surface-variant/70">
            Column changes are applied to the existing table with ALTER TABLE statements. Column renames are not inferred — use the dedicated Rename flow if needed.
          </p>
        )}
        {!edit && showSql && sql && (
          <pre className="font-code-md text-code-md text-primary bg-surface-container-low border border-outline-variant rounded-lg p-3 whitespace-pre-wrap break-words max-h-40 overflow-y-auto">
{sql}
          </pre>
        )}
      </div>
    </Modal>
  );
}