import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { createLazyFileRoute, useParams } from "@tanstack/react-router";
import { useDatabaseDetail, useStudioCreateTable, useStudioExec, useStudioQuery, useStudioSchemas, useStudioTable } from "../../hooks";
import type { StudioQueryResult, StudioExecResult } from "../../api/types";
import { Button } from "../../components/ui";
import { StudioSidebar } from "./-components/StudioSidebar";
import { SqlEditor, type SqlEditorApi } from "./-components/SqlEditor";
import { createEngine, type SqlEngine } from "../../studio-intelligence/engine";
import { getSnapshot, refreshSnapshot } from "../../studio-intelligence/schema";
import { DataGrid } from "./-components/DataGrid";
import { CreateTableModal } from "./-components/CreateTableModal";

interface Sel {
  schema: string;
  table: string;
}

type ResultsTab = "results" | "messages" | "plan";

const WRITE_RE = /^\s*(insert|update|delete|drop|create|alter|truncate|replace|grant|revoke|call|exec|execute|commit|rollback|set|begin)\b/i;

function qualifiedName(engine: string | undefined, schema: string, table: string): string {
  if (engine === "mysql" || engine === "mariadb") return `\`${schema}\`.\`${table}\``;
  if (engine === "mssql") return `[${schema}].[${table}]`;
  if (engine === "oracle") return `${schema.toUpperCase()}.${table.toUpperCase()}`;
  return `"${schema}"."${table}"`;
}

function selectSql(engine: string | undefined, sel: Sel): string {
  if (engine === "mongodb") return `db.${sel.table}.find().limit(200).toArray();`;
  if (engine === "redis") return `SCAN 0 COUNT 200`;
  return `SELECT * FROM ${qualifiedName(engine, sel.schema, sel.table)} LIMIT 200;`;
}

interface Tab {
  id: string;
  title: string;
  sql: string;
  result?: StudioQueryResult;
  execResult?: StudioExecResult;
  resultsTab: ResultsTab;
  active: boolean;
}

const tabStorageKey = (dbId: string) => `aether_studio_tabs_${dbId}`;

function freshTab(n: number): Tab {
  return { id: `tab_${Date.now()}_${n}`, title: `query_${String(n).padStart(2, "0")}.sql`, sql: "", resultsTab: "results", active: false };
}


// Statement splitter that respects single quotes, double quotes, backticks
// and PostgreSQL dollar-quoted strings, so semicolons inside strings/identifiers
// do not break the boundary detection.
function splitStatements(sql: string): { text: string; start: number; end: number }[] {
  const out: { text: string; start: number; end: number }[] = [];
  let start = 0;
  let i = 0;
  let quote: string | null = null;
  let dollar: string | null = null;
  const n = sql.length;
  while (i < n) {
    const ch = sql[i];
    if (dollar) {
      if (sql.startsWith(dollar, i)) {
        i += dollar.length;
        dollar = null;
        continue;
      }
    } else if (quote) {
      if (ch === quote) {
        if (quote === "'" && sql[i + 1] === "'") {
          i += 2;
          continue;
        }
        quote = null;
      }
    } else if (ch === "$") {
      const m = /^\$[A-Za-z_0-9]*\$/.exec(sql.slice(i));
      if (m) {
        dollar = m[0];
        i += m[0].length;
        continue;
      }
    } else if (ch === "'" || ch === '"' || ch === "`") {
      quote = ch;
    } else if (ch === ";") {
      out.push({ text: sql.slice(start, i + 1), start, end: i + 1 });
      start = i + 1;
    }
    i++;
  }
  if (start < n) {
    const tail = sql.slice(start);
    if (tail.trim() !== "") out.push({ text: tail, start, end: n });
  }
  return out;
}

function statementAtCursor(sql: string, cursorOffset: number): string {
  const stmts = splitStatements(sql);
  for (const s of stmts) {
    if (cursorOffset >= s.start && cursorOffset <= s.end) {
      return s.text.trim();
    }
  }
  const last = stmts[stmts.length - 1];
  return last ? last.text.trim() : sql.trim();
}

function StudioPage() {
  const queryClient = useQueryClient();
  const { dbId } = useParams({ strict: false }) as { dbId: string };
  const { data } = useDatabaseDetail(dbId);
  const query = useStudioQuery(dbId);
  const exec = useStudioExec(dbId);
  const createTable = useStudioCreateTable(dbId);
  const schemasQ = useStudioSchemas(dbId);

  const [tabs, setTabs] = useState<Tab[]>(() => {
    try {
      const saved = localStorage.getItem(tabStorageKey(dbId));
      if (saved) {
        const parsed = JSON.parse(saved) as Tab[];
        if (Array.isArray(parsed) && parsed.length > 0) {
          const restored: Tab[] = parsed.map((t) => ({ ...t, result: undefined, execResult: undefined, resultsTab: "results" as ResultsTab, active: false }));
          restored[restored.length - 1].active = true;
          return restored;
        }
      }
    } catch { /* ignore */ }
    return [freshTab(1)];
  });
  const [counter, setCounter] = useState(() => tabs.length + 1);
  const [activeObject, setActiveObject] = useState<Sel | null>(null);
  const [running, setRunning] = useState(false);
  const [editorH, setEditorH] = useState(340);
  const [showCreateTable, setShowCreateTable] = useState(false);
  const dragRef = useRef(false);
  const editorApiRef = useRef<SqlEditorApi | null>(null);
  const runHandlerRef = useRef<(() => void) | null>(null);
  const engineRef = useRef<SqlEngine | null>(null);
  if (engineRef.current === null) {
    engineRef.current = createEngine(dbId);
  }
  const runShortcutRef = useRef(false);

  useEffect(() => {
    if (!dbId) return;
    let active = true;
    (async () => {
      try {
        const snap = await getSnapshot(dbId);
        if (active) engineRef.current?.setSnapshot(snap);
      } catch {
        /* offline */
      }
    })();
    return () => {
      active = false;
    };
  }, [dbId]);

  const invalidateSchema = async () => {
    try {
      const snap = await refreshSnapshot(dbId);
      engineRef.current?.setSnapshot(snap);
    } catch {
      /* keep old */
    }
  };

  const onSchemaChanged = async () => {
    await invalidateSchema();
    queryClient.invalidateQueries({ queryKey: ["studio", dbId] });
  };

  const DDL_RE = /^\s*(create|alter|drop|truncate)/i;

  const activeTab = tabs.find((t) => t.active) ?? tabs[0];

  useEffect(() => {
    const persisted = tabs.map((t) => ({ id: t.id, title: t.title, sql: t.sql, active: t.active, resultsTab: t.resultsTab }));
    localStorage.setItem(tabStorageKey(dbId), JSON.stringify(persisted));
  }, [dbId, tabs]);

  const db = data?.database;
  const engine = db?.engine;
  const isRelational = !(engine === "mongodb" || engine === "redis");
  const schemas = schemasQ.data ?? [];

  const { data: activeDetail } = useStudioTable(dbId, activeObject?.schema ?? "", activeObject?.table ?? "");

  const runCurrent = () => {
    const api = editorApiRef.current;
    const text = activeTab.sql;
    if (!text.trim()) return;
    const selected = api?.getSelection();
    const sqlToRun = (selected ?? statementAtCursor(text, api?.getCursorOffset() ?? 0)).trim();
    if (!sqlToRun) return;
    void runQuery(activeTab.id, sqlToRun);
  };
  runHandlerRef.current = runCurrent;

  const runQuery = async (tabId: string, text: string) => {
    if (!text.trim()) return;
    setRunning(true);
    setTabs((ts) => ts.map((t) => (t.id === tabId ? { ...t, result: undefined, execResult: undefined } : t)));
    try {
      const isWrite = isRelational && WRITE_RE.test(text);
      if (isWrite) {
        const r = await exec.mutateAsync(text);
        setTabs((ts) => ts.map((t) => (t.id === tabId ? { ...t, execResult: r } : t)));
        engineRef.current?.recordQuery(text);
        if (DDL_RE.test(text)) void invalidateSchema();
      } else {
        const r = await query.mutateAsync(text);
        setTabs((ts) => ts.map((t) => (t.id === tabId ? { ...t, result: r } : t)));
        engineRef.current?.recordQuery(text);
      }
    } finally {
      setRunning(false);
    }
  };

  const openTab = (title: string, sql: string) => {
    const n = counter;
    setCounter(n + 1);
    const tab = freshTab(n);
    tab.title = title;
    tab.sql = sql;
    tab.active = true;
    setTabs((ts) => ts.map((t) => ({ ...t, active: false })).concat(tab));
    return tab;
  };

  const closeTab = (id: string) => {
    setTabs((ts) => {
      const idx = ts.findIndex((t) => t.id === id);
      if (idx === -1) return ts;
      const wasActive = ts[idx].active;
      const next = ts.filter((t) => t.id !== id);
      if (wasActive && next.length > 0) {
        next[Math.min(idx, next.length - 1)].active = true;
      }
      return next.length ? next : [freshTab(1)];
    });
  };

  const handleSelect = (sel: Sel) => {
    setActiveObject(sel);
    const sql = selectSql(engine, sel);
    const tab = openTab(`${sel.table}.sql`, sql);
    void runQuery(tab.id, sql);
  };

  const setActiveSql = (sql: string) => {
    setTabs((ts) => ts.map((t) => (t.id === activeTab.id ? { ...t, sql } : t)));
  };

  const setActiveResultsTab = (rt: ResultsTab) => {
    setTabs((ts) => ts.map((t) => (t.id === activeTab.id ? { ...t, resultsTab: rt } : t)));
  };

  const colTypes = activeDetail ? activeDetail.columns.map((c) => c.type) : undefined;

  const onDragMove = (e: React.MouseEvent) => {
    if (!dragRef.current) return;
    const panel = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const ratio = (e.clientY - panel.top) / panel.height;
    setEditorH(Math.min(Math.max(ratio * panel.height, 140), panel.height - 140));
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
        e.preventDefault();
        runCurrent();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  return (
    <div className="flex h-dvh w-full bg-background">
      <StudioSidebar dbId={dbId} selected={activeObject} onSelect={handleSelect} schemas={schemas} onSchemaChanged={onSchemaChanged} />

      <main className="flex-1 flex flex-col min-w-0 bg-background overflow-hidden relative">
        <div className="h-12 border-b border-outline-variant flex items-center px-md justify-between bg-background z-10">
          <div className="flex items-center gap-sm font-code-md text-[11px] text-on-surface-variant">
            <span className="hover:text-primary cursor-pointer transition-colors">{db?.name ?? "Database"}</span>
            {activeObject && (
              <>
                <span className="material-symbols-outlined text-[14px]">chevron_right</span>
                <span className="hover:text-primary cursor-pointer transition-colors">{activeObject.schema}</span>
                <span className="material-symbols-outlined text-[14px]">chevron_right</span>
                <span className="text-on-background">{activeObject.table}</span>
              </>
            )}
          </div>
          <div className="flex items-center gap-sm">
            <Button size="sm" variant="secondary" leftIcon="add_box" onClick={() => setShowCreateTable(true)}>
              Create Table
            </Button>
            <Button size="sm" leftIcon="play_arrow" loading={running} onClick={runCurrent}>
              Run Query
            </Button>
          </div>
        </div>

        <div className="flex-1 flex flex-col overflow-hidden">
          <div className="flex flex-col border-b border-outline-variant min-h-[120px]" style={{ height: editorH }}>
            <div className="flex bg-surface-container-lowest border-b border-outline-variant h-9 shrink-0 overflow-x-auto">
              {tabs.map((t) => (
                <div
                  key={t.id}
                  onClick={() => setTabs((ts) => ts.map((x) => ({ ...x, active: x.id === t.id })))}
                  className={`flex items-center gap-xs px-md border-r border-outline-variant font-code-md text-[11px] cursor-pointer select-none whitespace-nowrap ${
                    t.active ? "bg-surface-container-high border-t-2 border-t-primary text-primary" : "hover:bg-surface-container-high text-on-surface-variant"
                  }`}
                >
                  <span className="material-symbols-outlined text-[14px]">terminal</span>
                  {t.title}
                  <span
                    role="button"
                    aria-label={`Close ${t.title}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      closeTab(t.id);
                    }}
                    className="material-symbols-outlined text-[14px] hover:text-error ml-1"
                  >
                    close
                  </span>
                </div>
              ))}
              <button
                onClick={() => openTab(`query_${String(counter).padStart(2, "0")}.sql`, "")}
                className="flex items-center gap-xs px-md font-code-md text-[11px] text-on-surface-variant hover:text-primary transition-colors"
                aria-label="New tab"
              >
                <span className="material-symbols-outlined text-[14px]">add</span>
              </button>
            </div>
            <div className="flex-1 min-h-0">
              <SqlEditor value={activeTab.sql} onChange={setActiveSql} language={isRelational ? "sql" : engine === "mongodb" ? "javascript" : "shell"} height="100%" apiRef={editorApiRef} engineRef={engineRef} runHandlerRef={runHandlerRef} />
            </div>
          </div>

          <div
            className="h-1.5 cursor-row-resize bg-surface-container-low hover:bg-primary/50 transition-colors shrink-0"
            onMouseDown={() => (dragRef.current = true)}
            onMouseUp={() => (dragRef.current = false)}
            onMouseMove={onDragMove}
            onMouseLeave={() => (dragRef.current = false)}
          />

          <div className="flex-1 flex flex-col min-h-[120px]">
            <div className="flex bg-surface-container-lowest border-b border-outline-variant h-9 shrink-0">
              {(
                [
                  { id: "results", label: "Results", icon: "table_rows" },
                  { id: "messages", label: "Messages", icon: "chat" },
                  { id: "plan", label: "Execution Plan", icon: "account_tree" },
                ] as { id: ResultsTab; label: string; icon: string }[]
              ).map((t) => (
                <button
                  key={t.id}
                  onClick={() => setActiveResultsTab(t.id)}
                  className={`flex items-center gap-xs px-md border-b-2 font-label-caps text-label-caps transition-colors ${
                    activeTab.resultsTab === t.id ? "border-primary text-primary" : "border-transparent text-on-surface-variant hover:text-on-surface"
                  }`}
                >
                  <span className="material-symbols-outlined text-[14px]">{t.icon}</span>
                  {t.label}
                </button>
              ))}
              <div className="ml-auto flex items-center px-sm gap-sm text-[11px] text-on-surface-variant font-code-md">
                {activeTab.result && !activeTab.result.error && (
                  <>
                    <span className="flex items-center gap-1">
                      <span className="w-2 h-2 rounded-full bg-[#82aaff] inline-block" /> Query OK
                    </span>
                    <span>Time: {activeTab.result.duration_ms}ms</span>
                    <span>Rows: {activeTab.result.row_count.toLocaleString()}</span>
                  </>
                )}
                {running && <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-[#f78c6c] animate-pulse inline-block" /> Running...</span>}
              </div>
            </div>

            <div className="flex-1 min-h-0 overflow-hidden">
              {activeTab.resultsTab === "results" && (
                <>
                  {query.isError && (
                    <div className="m-md p-md rounded-lg border border-error/40 bg-error/10 font-code-md text-code-md text-error">
                      {(query.error as Error)?.message}
                    </div>
                  )}
                  {exec.isError && (
                    <div className="m-md p-md rounded-lg border border-error/40 bg-error/10 font-code-md text-code-md text-error">
                      {(exec.error as Error)?.message}
                    </div>
                  )}
                  {activeTab.execResult && (
                    <div className="m-md p-md rounded-lg bg-surface-container-high border border-outline-variant font-code-md text-code-md text-on-surface">
                      <span className="text-primary">{activeTab.execResult.command_tag || "OK"}</span> — {activeTab.execResult.message} ({activeTab.execResult.duration_ms} ms)
                    </div>
                  )}
                  <div className="h-[calc(100%-1px)]">
                    <DataGrid result={activeTab.result} empty="Run a query or select a table to see data." types={colTypes} />
                  </div>
                </>
              )}
              {activeTab.resultsTab === "messages" && (
                <div className="p-md font-code-md text-code-md text-on-surface-variant">
                  {activeTab.execResult ? `${activeTab.execResult.command_tag || "OK"} — ${activeTab.execResult.message}` : "No messages. Run a query to see output here."}
                </div>
              )}
              {activeTab.resultsTab === "plan" && (
                <div className="p-md font-code-md text-code-md text-on-surface-variant">
                  Execution plan not available for this result. Prefix your query with <span className="text-primary">EXPLAIN</span> to inspect it.
                </div>
              )}
            </div>
          </div>
        </div>
      </main>

      <CreateTableModal
        open={showCreateTable}
        onClose={() => setShowCreateTable(false)}
        engine={engine}
        schemas={schemas}
        onCreate={async (payload) => {
          const { sql, table, schema, columns } = payload;
          if (!sql) return;
          const res = await createTable.mutateAsync({ table, schema, columns });
          const tab = openTab("create_table.sql", sql);
          setTabs((ts) => ts.map((t) => (t.id === tab.id ? { ...t, execResult: res } : t)));
        }}
      />
    </div>
  );
}

export const Route = createLazyFileRoute("/studio/$dbId/")({
  component: StudioPage,
});