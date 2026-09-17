import {
  useEffect,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { createLazyFileRoute, useParams } from "@tanstack/react-router";
import {
  ArrowRight,
  ChartLine,
  ChatCircle,
  Code,
  Database,
  List,
  Play,
  Plus,
  Table,
  TerminalWindow,
  X,
} from "@phosphor-icons/react";
import {
  useDatabaseDetail,
  useStudioCreateTable,
  useStudioExec,
  useStudioQuery,
  useStudioSchemas,
  useStudioTable,
} from "../../hooks";
import type { StudioQueryResult, StudioExecResult } from "../../api/types";
import { Button } from "@aether/design-system";
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

const WRITE_RE =
  /^\s*(insert|update|delete|drop|create|alter|truncate|replace|grant|revoke|call|exec|execute|commit|rollback|set|begin)\b/i;

function qualifiedName(
  engine: string | undefined,
  schema: string,
  table: string,
): string {
  if (engine === "mysql" || engine === "mariadb")
    return `\`${schema}\`.\`${table}\``;
  if (engine === "mssql") return `[${schema}].[${table}]`;
  if (engine === "oracle")
    return `${schema.toUpperCase()}.${table.toUpperCase()}`;
  return `"${schema}"."${table}"`;
}

function selectSql(engine: string | undefined, sel: Sel): string {
  if (engine === "mongodb")
    return `db.${sel.table}.find().limit(200).toArray();`;
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
  return {
    id: `tab_${Date.now()}_${n}`,
    title: `query_${String(n).padStart(2, "0")}.sql`,
    sql: "",
    resultsTab: "results",
    active: false,
  };
}

// Statement splitter that respects single quotes, double quotes, backticks
// and PostgreSQL dollar-quoted strings, so semicolons inside strings/identifiers
// do not break the boundary detection.
function splitStatements(
  sql: string,
): { text: string; start: number; end: number }[] {
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
          const restored: Tab[] = parsed.map((t) => ({
            ...t,
            result: undefined,
            execResult: undefined,
            resultsTab: "results" as ResultsTab,
            active: false,
          }));
          restored[restored.length - 1].active = true;
          return restored;
        }
      }
    } catch {
      /* ignore */
    }
    return [freshTab(1)];
  });
  const [counter, setCounter] = useState(() => tabs.length + 1);
  const [activeObject, setActiveObject] = useState<Sel | null>(null);
  const [running, setRunning] = useState(false);
  const [editorH, setEditorH] = useState(340);
  const [showCreateTable, setShowCreateTable] = useState(false);
  const [explorerOpen, setExplorerOpen] = useState(false);
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

  const DDL_RE = /^\s*(create|alter|drop|truncate)\b/i;

  const activeTab = tabs.find((t) => t.active) ?? tabs[0];

  useEffect(() => {
    const persisted = tabs.map((t) => ({
      id: t.id,
      title: t.title,
      sql: t.sql,
      active: t.active,
      resultsTab: t.resultsTab,
    }));
    localStorage.setItem(tabStorageKey(dbId), JSON.stringify(persisted));
  }, [dbId, tabs]);

  const db = data?.database;
  const engine = db?.engine;
  const isRelational = !(engine === "mongodb" || engine === "redis");
  const schemas = schemasQ.data ?? [];

  const { data: activeDetail } = useStudioTable(
    dbId,
    activeObject?.schema ?? "",
    activeObject?.table ?? "",
  );

  const runCurrent = () => {
    const api = editorApiRef.current;
    const text = activeTab.sql;
    if (!text.trim()) return;
    const selected = api?.getSelection();
    const sqlToRun = (
      selected ?? statementAtCursor(text, api?.getCursorOffset() ?? 0)
    ).trim();
    if (!sqlToRun) return;
    void runQuery(activeTab.id, sqlToRun);
  };
  runHandlerRef.current = runCurrent;

  const runQuery = async (tabId: string, text: string) => {
    if (!text.trim()) return;
    setRunning(true);
    setTabs((ts) =>
      ts.map((t) =>
        t.id === tabId ? { ...t, result: undefined, execResult: undefined } : t,
      ),
    );
    try {
      const isWrite = isRelational && WRITE_RE.test(text);
      if (isWrite) {
        const r = await exec.mutateAsync(text);
        setTabs((ts) =>
          ts.map((t) => (t.id === tabId ? { ...t, execResult: r } : t)),
        );
        engineRef.current?.recordQuery(text);
        if (DDL_RE.test(text)) void invalidateSchema();
      } else {
        const r = await query.mutateAsync(text);
        setTabs((ts) =>
          ts.map((t) => (t.id === tabId ? { ...t, result: r } : t)),
        );
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
      if (next.length) return next;
      const fallback = freshTab(1);
      fallback.active = true;
      return [fallback];
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
    setTabs((ts) =>
      ts.map((t) => (t.id === activeTab.id ? { ...t, resultsTab: rt } : t)),
    );
  };

  const colTypes = activeDetail
    ? activeDetail.columns.map((c) => c.type)
    : undefined;

  const queryTabs = [
    { id: "results", label: "Results", icon: Table },
    { id: "messages", label: "Messages", icon: ChatCircle },
    { id: "plan", label: "Execution Plan", icon: ChartLine },
  ] as { id: ResultsTab; label: string; icon: typeof Table }[];
  const studioContentRef = useRef<HTMLDivElement>(null);
  const selectQueryTab = (id: string) => {
    setTabs((ts) => ts.map((t) => ({ ...t, active: t.id === id })));
  };
  const handleQueryTabKeyDown = (
    event: ReactKeyboardEvent<HTMLButtonElement>,
    index: number,
  ) => {
    if (
      event.key !== "ArrowRight" &&
      event.key !== "ArrowLeft" &&
      event.key !== "Home" &&
      event.key !== "End"
    )
      return;
    event.preventDefault();
    const nextIndex =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? tabs.length - 1
          : (index + (event.key === "ArrowRight" ? 1 : -1) + tabs.length) %
            tabs.length;
    const nextTab = tabs[nextIndex];
    selectQueryTab(nextTab.id);
    document.getElementById(`query-tab-${nextTab.id}`)?.focus();
  };
  const handleResultsTabKeyDown = (
    event: ReactKeyboardEvent<HTMLButtonElement>,
    index: number,
  ) => {
    if (
      event.key !== "ArrowRight" &&
      event.key !== "ArrowLeft" &&
      event.key !== "Home" &&
      event.key !== "End"
    )
      return;
    event.preventDefault();
    const nextIndex =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? queryTabs.length - 1
          : (index + (event.key === "ArrowRight" ? 1 : -1) + queryTabs.length) %
            queryTabs.length;
    const nextTab = queryTabs[nextIndex];
    setActiveResultsTab(nextTab.id);
    document.getElementById(`results-tab-${nextTab.id}`)?.focus();
  };
  const onDragMove = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (!dragRef.current || !studioContentRef.current) return;
    const panel = studioContentRef.current.getBoundingClientRect();
    const maxHeight = Math.max(140, panel.height - 140);
    setEditorH(Math.min(Math.max(e.clientY - panel.top, 140), maxHeight));
  };
  const onResizeKeyDown = (e: ReactKeyboardEvent<HTMLDivElement>) => {
    if (
      !["ArrowUp", "ArrowDown", "Home", "End"].includes(e.key) ||
      !studioContentRef.current
    )
      return;
    e.preventDefault();
    const maxHeight = Math.max(
      140,
      studioContentRef.current.clientHeight - 140,
    );
    const next =
      e.key === "Home"
        ? 140
        : e.key === "End"
          ? maxHeight
          : editorH + (e.key === "ArrowDown" ? 24 : -24);
    setEditorH(Math.min(Math.max(next, 140), maxHeight));
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
    <div className="flex h-dvh w-full bg-surface-container-low">
      <StudioSidebar
        dbId={dbId}
        selected={activeObject}
        onSelect={(selection) => {
          handleSelect(selection);
          setExplorerOpen(false);
        }}
        schemas={schemas}
        onSchemaChanged={onSchemaChanged}
        mobileOpen={explorerOpen}
        onMobileClose={() => setExplorerOpen(false)}
      />
      {explorerOpen ? (
        <button
          type="button"
          aria-label="Close object explorer"
          onClick={() => setExplorerOpen(false)}
          className="fixed inset-0 z-30 bg-black/60 backdrop-blur-sm sm:hidden"
        />
      ) : null}

      <main className="relative flex min-w-0 flex-1 flex-col overflow-hidden bg-surface-container-low">
        <div className="z-10 flex h-14 items-center justify-between border-b border-outline-variant bg-surface-container-low/90 px-lg backdrop-blur-xl">
          <div className="flex min-w-0 items-center gap-sm font-code-md text-[11px] text-on-surface-variant">
            <button
              type="button"
              onClick={() => setExplorerOpen(true)}
              className="rounded-lg p-1.5 text-on-surface-variant outline-none transition-colors hover:bg-surface-container-high hover:text-primary focus-visible:ring-2 focus-visible:ring-ring sm:hidden"
              aria-label="Open object explorer"
            >
              <List size={16} aria-hidden="true" />
            </button>
            <span className="hover:text-primary cursor-pointer transition-colors">
              {db?.name ?? "Database"}
            </span>
            {activeObject && (
              <>
                <ArrowRight size={14} aria-hidden="true" />
                <span className="hover:text-primary cursor-pointer transition-colors">
                  {activeObject.schema}
                </span>
                <ArrowRight size={14} aria-hidden="true" />
                <span className="text-on-background">{activeObject.table}</span>
              </>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-sm">
            <Button
              size="sm"
              variant="secondary"
              onClick={() => setShowCreateTable(true)}
            >
              <Plus size={16} aria-hidden="true" />
              <span className="max-sm:hidden">Create Table</span>
            </Button>
            <Button size="sm" loading={running} onClick={runCurrent}>
              <Play size={16} aria-hidden="true" />
              <span className="max-sm:hidden">Run Query</span>
            </Button>
          </div>
        </div>

        <div
          ref={studioContentRef}
          className="flex-1 flex flex-col overflow-hidden"
        >
          <div
            className="flex flex-col border-b border-outline-variant min-h-[120px]"
            style={{ height: editorH }}
          >
            <div
              className="flex h-10 shrink-0 overflow-x-auto rounded-t-xl border-b border-outline-variant bg-surface-container-lowest"
              role="tablist"
              aria-label="Query tabs"
            >
              {tabs.map((t) => (
                <div
                  key={t.id}
                  className={`flex items-center border-r border-outline-variant ${t.active ? "bg-surface-container-high" : "hover:bg-surface-container-high"}`}
                >
                  <button
                    type="button"
                    role="tab"
                    id={`query-tab-${t.id}`}
                    aria-controls="query-editor-panel"
                    aria-selected={t.active}
                    tabIndex={t.active ? 0 : -1}
                    onClick={() => selectQueryTab(t.id)}
                    onKeyDown={(event) =>
                      handleQueryTabKeyDown(event, tabs.indexOf(t))
                    }
                    className={`flex items-center gap-xs px-md py-2 font-code-md text-[11px] outline-none ${t.active ? "border-t-2 border-t-primary text-primary" : "border-t-2 border-t-transparent text-on-surface-variant"} focus-visible:ring-2 focus-visible:ring-ring`}
                  >
                    <TerminalWindow size={14} aria-hidden="true" />
                    {t.title}
                  </button>
                  <button
                    type="button"
                    aria-label={`Close ${t.title}`}
                    onClick={() => closeTab(t.id)}
                    className="mr-1 rounded p-1 text-[14px] text-on-surface-variant outline-none hover:text-error focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    <X size={14} aria-hidden="true" />
                  </button>
                </div>
              ))}
              <button
                type="button"
                onClick={() =>
                  openTab(`query_${String(counter).padStart(2, "0")}.sql`, "")
                }
                className="flex items-center gap-xs px-md font-code-md text-[11px] text-on-surface-variant hover:text-primary transition-colors"
                aria-label="New tab"
              >
                <Plus size={14} aria-hidden="true" />
              </button>
            </div>
            <div
              id="query-editor-panel"
              role="tabpanel"
              aria-labelledby={`query-tab-${activeTab.id}`}
              tabIndex={0}
              className="flex-1 min-h-0 outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <SqlEditor
                value={activeTab.sql}
                onChange={setActiveSql}
                language={
                  isRelational
                    ? "sql"
                    : engine === "mongodb"
                      ? "javascript"
                      : "shell"
                }
                height="100%"
                apiRef={editorApiRef}
                engineRef={engineRef}
                runHandlerRef={runHandlerRef}
              />
            </div>
          </div>

          <div
            role="separator"
            aria-label="Resize query editor"
            aria-orientation="horizontal"
            aria-valuemin={140}
            aria-valuemax={Math.max(
              140,
              (studioContentRef.current?.clientHeight ?? editorH + 140) - 140,
            )}
            aria-valuenow={editorH}
            tabIndex={0}
            className="h-1.5 shrink-0 cursor-row-resize bg-surface-container-low outline-none transition-colors hover:bg-primary/50 focus-visible:bg-primary/50 focus-visible:ring-2 focus-visible:ring-ring"
            onPointerDown={(event) => {
              dragRef.current = true;
              event.currentTarget.setPointerCapture(event.pointerId);
            }}
            onPointerUp={(event) => {
              dragRef.current = false;
              if (event.currentTarget.hasPointerCapture(event.pointerId))
                event.currentTarget.releasePointerCapture(event.pointerId);
            }}
            onPointerCancel={() => (dragRef.current = false)}
            onPointerMove={onDragMove}
            onKeyDown={onResizeKeyDown}
          />

          <div className="flex-1 flex flex-col min-h-[120px]">
            <div
              className="flex h-10 shrink-0 rounded-t-xl border-b border-outline-variant bg-surface-container-lowest"
              role="tablist"
              aria-label="Query output"
            >
              {queryTabs.map((t) => (
                <button
                  type="button"
                  key={t.id}
                  onClick={() => setActiveResultsTab(t.id)}
                  role="tab"
                  id={`results-tab-${t.id}`}
                  aria-controls="results-panel"
                  aria-selected={activeTab.resultsTab === t.id}
                  tabIndex={activeTab.resultsTab === t.id ? 0 : -1}
                  onKeyDown={(event) =>
                    handleResultsTabKeyDown(event, queryTabs.indexOf(t))
                  }
                  className={`flex items-center gap-xs px-md border-b-2 font-label-caps text-label-caps outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring ${
                    activeTab.resultsTab === t.id
                      ? "border-primary text-primary"
                      : "border-transparent text-on-surface-variant hover:text-on-surface"
                  }`}
                >
                  <t.icon size={14} aria-hidden="true" />
                  {t.label}
                </button>
              ))}
              <div className="ml-auto flex items-center px-sm gap-sm text-[11px] text-on-surface-variant font-code-md">
                {activeTab.result && !activeTab.result.error && (
                  <>
                    <span className="flex items-center gap-1">
                      <span className="size-2 rounded-full bg-status-success" />{" "}
                      Query OK
                    </span>
                    <span>Time: {activeTab.result.duration_ms}ms</span>
                    <span>
                      Rows: {activeTab.result.row_count.toLocaleString()}
                    </span>
                  </>
                )}
                {running && (
                  <span className="flex items-center gap-1" role="status">
                    <span className="size-2 rounded-full bg-status-warning" />{" "}
                    Running...
                  </span>
                )}
              </div>
            </div>

            <div
              id="results-panel"
              role="tabpanel"
              aria-labelledby={`results-tab-${activeTab.resultsTab}`}
              tabIndex={0}
              className="flex-1 min-h-0 overflow-hidden outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              {activeTab.resultsTab === "results" && (
                <>
                  {query.isError && (
                    <div className="m-md rounded-xl border border-error/40 bg-error/10 p-md font-code-md text-code-md text-error">
                      {(query.error as Error)?.message}
                    </div>
                  )}
                  {exec.isError && (
                    <div className="m-md rounded-xl border border-error/40 bg-error/10 p-md font-code-md text-code-md text-error">
                      {(exec.error as Error)?.message}
                    </div>
                  )}
                  {activeTab.execResult && (
                    <div className="m-md rounded-xl border border-outline-variant bg-surface-container-high p-md font-code-md text-code-md text-on-surface">
                      <span className="text-primary">
                        {activeTab.execResult.command_tag || "OK"}
                      </span>{" "}
                      — {activeTab.execResult.message} (
                      {activeTab.execResult.duration_ms} ms)
                    </div>
                  )}
                  <div className="h-[calc(100%-1px)]">
                    <DataGrid
                      result={activeTab.result}
                      empty="Run a query or select a table to see data."
                      types={colTypes}
                    />
                  </div>
                </>
              )}
              {activeTab.resultsTab === "messages" && (
                <div className="p-md font-code-md text-code-md text-on-surface-variant">
                  {activeTab.execResult
                    ? `${activeTab.execResult.command_tag || "OK"} — ${activeTab.execResult.message}`
                    : "No messages. Run a query to see output here."}
                </div>
              )}
              {activeTab.resultsTab === "plan" && (
                <div className="p-md font-code-md text-code-md text-on-surface-variant">
                  Execution plan not available for this result. Prefix your
                  query with <span className="text-primary">EXPLAIN</span> to
                  inspect it.
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
          setTabs((ts) =>
            ts.map((t) => (t.id === tab.id ? { ...t, execResult: res } : t)),
          );
        }}
      />
    </div>
  );
}

export const Route = createLazyFileRoute("/studio/$dbId/")({
  component: StudioPage,
});
