import { useRef, useState } from "react";
import { createLazyFileRoute, useParams } from "@tanstack/react-router";
import {
  useDatabaseDetail,
  useStudioQuery,
  useStudioExec,
  useStudioTable,
} from "../../../../hooks";
import type { StudioQueryResult, StudioExecResult } from "../../../../api/types";
import { Button } from "../../../../components/ui";
import { StudioSidebar } from "./-components/StudioSidebar";
import { SqlEditor } from "./-components/SqlEditor";
import { DataGrid } from "./-components/DataGrid";

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

function StudioPage() {
  const { dbId } = useParams({ strict: false }) as { dbId: string };
  const { data } = useDatabaseDetail(dbId);
  const query = useStudioQuery(dbId);
  const exec = useStudioExec(dbId);

  const [sql, setSql] = useState("");
  const [activeObject, setActiveObject] = useState<Sel | null>(null);
  const [result, setResult] = useState<StudioQueryResult | undefined>(undefined);
  const [execResult, setExecResult] = useState<StudioExecResult | undefined>(undefined);
  const [resultTypes, setResultTypes] = useState<string[] | undefined>(undefined);
  const [running, setRunning] = useState(false);
  const [resultsTab, setResultsTab] = useState<ResultsTab>("results");
  const [editorH, setEditorH] = useState(340);
  const dragRef = useRef(false);

  const db = data?.database;
  const engine = db?.engine;
  const isRelational = !(engine === "mongodb" || engine === "redis");

  const { data: activeDetail } = useStudioTable(dbId, activeObject?.schema ?? "", activeObject?.table ?? "");

  const runQuery = async (text: string, types?: string[]) => {
    if (!text.trim()) return;
    setRunning(true);
    setExecResult(undefined);
    setResult(undefined);
    setResultTypes(types);
    try {
      const isWrite = isRelational && WRITE_RE.test(text);
      if (isWrite) {
        setExecResult(await exec.mutateAsync(text));
      } else {
        setResult(await query.mutateAsync(text));
      }
    } finally {
      setRunning(false);
    }
  };

  const handleSelect = (sel: Sel) => {
    setActiveObject(sel);
    setResultsTab("results");
    const types = engine === "mongodb" || engine === "redis" ? undefined : undefined;
    void runQuery(selectSql(engine, sel), types);
  };

  const colTypes = resultTypes ?? (activeDetail ? activeDetail.columns.map((c) => c.type) : undefined);

  const onDragMove = (e: React.MouseEvent) => {
    if (!dragRef.current) return;
    const panel = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const ratio = (e.clientY - panel.top) / panel.height;
    setEditorH(Math.min(Math.max(ratio * panel.height, 140), panel.height - 140));
  };

  return (
    <div className="flex h-[calc(100dvh-7.5rem)] -mx-margin-desktop">
      <StudioSidebar dbId={dbId} selected={activeObject} onSelect={handleSelect} />

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
          <Button size="sm" leftIcon="play_arrow" loading={running} onClick={() => void runQuery(sql, activeObject ? colTypes : undefined)}>
            Run Query
          </Button>
        </div>

        <div className="flex-1 flex flex-col overflow-hidden">
          <div className="flex flex-col border-b border-outline-variant min-h-[120px]" style={{ height: editorH }}>
            <div className="flex bg-surface-container-lowest border-b border-outline-variant h-9 shrink-0">
              <div className="flex items-center gap-xs px-md border-r border-outline-variant bg-surface-container-high border-t-2 border-t-primary text-primary font-code-md text-[11px]">
                <span className="material-symbols-outlined text-[14px]">terminal</span>
                query_01.sql
                <span className="material-symbols-outlined text-[14px] hover:text-error ml-sm">close</span>
              </div>
              <div className="flex items-center gap-xs px-md border-r border-outline-variant hover:bg-surface-container-high text-on-surface-variant font-code-md text-[11px] cursor-pointer transition-colors border-t-2 border-t-transparent">
                <span className="material-symbols-outlined text-[14px]">terminal</span>
                create_table.sql
              </div>
            </div>
            <div className="flex-1 min-h-0">
              <SqlEditor value={sql} onChange={setSql} language={isRelational ? "sql" : engine === "mongodb" ? "javascript" : "shell"} height="100%" />
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
                  onClick={() => setResultsTab(t.id)}
                  className={`flex items-center gap-xs px-md border-b-2 font-label-caps text-label-caps transition-colors ${
                    resultsTab === t.id ? "border-primary text-primary" : "border-transparent text-on-surface-variant hover:text-on-surface"
                  }`}
                >
                  <span className="material-symbols-outlined text-[14px]">{t.icon}</span>
                  {t.label}
                </button>
              ))}
              <div className="ml-auto flex items-center px-sm gap-sm text-[11px] text-on-surface-variant font-code-md">
                {result && !result.error && (
                  <>
                    <span className="flex items-center gap-1">
                      <span className="w-2 h-2 rounded-full bg-[#82aaff] inline-block" /> Query OK
                    </span>
                    <span>Time: {result.duration_ms}ms</span>
                    <span>Rows: {result.row_count.toLocaleString()}</span>
                  </>
                )}
                {running && <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-[#f78c6c] animate-pulse inline-block" /> Running...</span>}
              </div>
            </div>

            <div className="flex-1 min-h-0 overflow-hidden">
              {resultsTab === "results" && (
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
                  {execResult && (
                    <div className="m-md p-md rounded-lg bg-surface-container-high border border-outline-variant font-code-md text-code-md text-on-surface">
                      <span className="text-primary">{execResult.command_tag || "OK"}</span> — {execResult.message} ({execResult.duration_ms} ms)
                    </div>
                  )}
                  <div className="h-[calc(100%-1px)]">
                    <DataGrid result={result} empty="Run a query or select a table to see data." types={colTypes} />
                  </div>
                </>
              )}
              {resultsTab === "messages" && (
                <div className="p-md font-code-md text-code-md text-on-surface-variant">
                  {execResult ? `${execResult.command_tag || "OK"} — ${execResult.message}` : "No messages. Run a query to see output here."}
                </div>
              )}
              {resultsTab === "plan" && (
                <div className="p-md font-code-md text-code-md text-on-surface-variant">
                  Execution plan not available for this result. Prefix your query with <span className="text-primary">EXPLAIN</span> to inspect it.
                </div>
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

export const Route = createLazyFileRoute("/_shell/databases/$dbId/studio/")({
  component: StudioPage,
});