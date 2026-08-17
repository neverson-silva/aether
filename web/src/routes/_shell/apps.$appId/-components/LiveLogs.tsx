import { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { usePresence } from "../../../../hooks";
import { getServer } from "../../../../api/client";

const ANSI_RE = /\u001b\[[0-9;]*[A-Za-z]/g;
const TIMESTAMP_RE = /^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?/;
const TAG_RE = /^\[([a-z_\-]+)\]/i;

export function stripANSI(line: string): string {
  return line.replace(ANSI_RE, "");
}

export function tryPrettyJSON(line: string): string | null {
  const t = line.trim();
  if (!t.startsWith("{") && !t.startsWith("[")) return null;
  try {
    return JSON.stringify(JSON.parse(t), null, 2);
  } catch {
    return null;
  }
}

type LogRow = { id: number; text: string; json: string | null; ts: string; level: string; tag: string };

let ROW_ID = 0;

const TAG_COLORS: Record<string, string> = {
  deploy: "text-[#22d3ee]",
  build: "text-[#fbbf24]",
  plan: "text-[#e879f9]",
  health: "text-[#4ade80]",
  scheduler: "text-[#60a5fa]",
  logs: "text-on-surface-variant",
  error: "text-error",
};

const LEVEL_COLOR: Record<string, string> = {
  error: "text-error",
  warn: "text-[#fbbf24]",
  info: "text-[#60a5fa]",
  debug: "text-on-surface-variant",
};

export function classify(line: string): LogRow {
  const text = stripANSI(line);
  const json = tryPrettyJSON(text);
  const tsMatch = text.match(TIMESTAMP_RE);
  const tagMatch = text.match(TAG_RE);
  const lower = text.toLowerCase();
  let level = "";
  if (/\b(error|failed|falhou|fatal|panic|exception|traceback)\b/.test(lower)) level = "error";
  else if (/\b(warn|warning)\b/.test(lower)) level = "warn";
  else if (/\b(debug)\b/.test(lower)) level = "debug";
  else if (/\b(ok|ready|conclu|started|listening|running|healthy)\b/.test(lower)) level = "info";
  ROW_ID += 1;
  return { id: ROW_ID, text, json, ts: tsMatch ? tsMatch[0] : "", level, tag: tagMatch ? tagMatch[1].toLowerCase() : "" };
}

export function LogLine({ row }: { row: LogRow }) {
  if (row.json) {
    return (
      <pre className="whitespace-pre-wrap break-all text-[#4ade80]/90">{row.json}</pre>
    );
  }
  const t = row.text;
  const tsLen = row.ts.length;
  let head = tsLen ? t.slice(0, tsLen) : "";
  let rest = tsLen ? t.slice(tsLen) : t;
  let tag = "";
  const tagMatch = rest.match(TAG_RE);
  if (tagMatch) {
    tag = tagMatch[0];
    rest = rest.slice(tag.length);
  }
  return (
    <span className="whitespace-pre-wrap break-all leading-relaxed">
      {head && <span className="text-on-surface-variant/40">{head}</span>}
      {tag && (
        <span className={`${TAG_COLORS[row.tag] ?? "text-on-surface-variant/70"} font-semibold`}>{tag}</span>
      )}
      <span className={LEVEL_COLOR[row.level] ?? "text-[#d1d5db]"}>
        {row.level === "error" ? `✗${rest}` : rest}
      </span>
    </span>
  );
}

export function LiveLogs({ appId }: { appId: string }) {
  const [rows, setRows] = useState<LogRow[]>([]);
  const [follow, setFollow] = useState(true);
  const [paused, setPaused] = useState(false);
  const [filter, setFilter] = useState("");
  const [regexMode, setRegexMode] = useState(false);
  const [levelFilter, setLevelFilter] = useState("");
  const [jsonOnly, setJsonOnly] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const atBottom = useRef(true);
  const viewers = usePresence("app:" + appId + ":logs");

  useEffect(() => {
    const server = getServer();
    const es = new EventSource(server + "/api/v1/apps/" + appId + "/logs?follow=1", { withCredentials: true });
    es.onmessage = (ev) => {
      const chunks = ev.data.split("\n").filter((c: string) => c.trim() !== "");
      const newRows = chunks.map((c: string) => classify(c));
      setRows((prev) => [...prev.slice(-1000), ...newRows].slice(-1500));
    };
    es.onerror = () => {
      /* reconexão automática do EventSource */
    };
    return () => es.close();
  }, [appId]);

  const scrollToBottom = useCallback(() => {
    ref.current?.scrollTo({ top: ref.current.scrollHeight, behavior: "smooth" });
  }, []);

  useEffect(() => {
    if (follow) scrollToBottom();
  }, [rows, follow, scrollToBottom]);

  const onScroll = () => {
    const el = ref.current;
    if (!el) return;
    atBottom.current = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    if (atBottom.current && paused) setPaused(false);
  };

  const filtered = useMemo(() => {
    let out = rows;
    if (levelFilter) out = out.filter((r) => r.level === levelFilter);
    if (jsonOnly) out = out.filter((r) => !!r.json);
    if (filter) {
      try {
        const rx = new RegExp(regexMode ? filter : filter.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "i");
        out = out.filter((r) => rx.test(r.text));
      } catch {
        out = out.filter((r) => r.text.toLowerCase().includes(filter.toLowerCase()));
      }
    }
    return out;
  }, [rows, filter, regexMode, levelFilter, jsonOnly]);

  const exportLogs = () => {
    const blob = new Blob([rows.map((r) => r.text).join("\n")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `app-${appId}.log`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const toggleFollow = () => {
    setFollow((f) => !f);
    if (!follow) {
      setPaused(false);
      scrollToBottom();
    }
  };

  return (
    <div className="flex flex-col h-full min-h-[320px]">
      <div className="flex items-center gap-xs px-md py-2 bg-[#050505] border-b border-[#1a1a1a] rounded-t-lg">
        <div className="flex gap-1.5 mr-sm">
          <span className="w-2.5 h-2.5 rounded-full bg-[#ff5f57]" />
          <span className="w-2.5 h-2.5 rounded-full bg-[#febc2e]" />
          <span className="w-2.5 h-2.5 rounded-full bg-[#28c840]" />
        </div>
        <span className="font-code-md text-[11px] text-on-surface-variant/60">aether · live logs</span>
        {follow && <span className="rt-live-dot" title="receiving live events" />}
        <div className="flex-1" />
        {viewers > 0 && (
          <span className="px-2 py-0.5 rounded font-code-md text-[11px] bg-surface-container-high text-on-surface-variant">
            👁 {viewers} {viewers === 1 ? "viewer" : "viewers"}
          </span>
        )}
        <button onClick={toggleFollow} className={`px-2 py-0.5 rounded font-code-md text-[11px] ${follow ? "bg-[#28c840]/15 text-[#28c840]" : "bg-surface-container-high text-on-surface-variant"}`}>
          {follow ? "● Live" : "○ Paused"}
        </button>
      </div>
      <div className="flex items-center gap-sm flex-wrap px-md py-2 bg-[#0a0a0a] border-b border-[#1a1a1a]">
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder={regexMode ? "Regex filter..." : "Filter logs..."}
          className="w-48 bg-[#0f0f0f] border border-[#262626] rounded px-2 py-1 font-code-md text-[11px] text-on-surface placeholder:text-on-surface-variant/40 focus:border-[#4a9eff] focus:outline-none"
        />
        <label className="flex items-center gap-1 cursor-pointer select-none">
          <input type="checkbox" checked={regexMode} onChange={(e) => setRegexMode(e.target.checked)} className="w-3 h-3 rounded-sm bg-surface border-outline-variant text-primary" />
          <span className="font-code-md text-[11px] text-on-surface-variant">regex</span>
        </label>
        <select
          value={levelFilter}
          onChange={(e) => setLevelFilter(e.target.value)}
          className="bg-[#0f0f0f] border border-[#262626] rounded px-2 py-1 font-code-md text-[11px] text-on-surface focus:border-[#4a9eff] focus:outline-none"
        >
          <option value="">All levels</option>
          <option value="error">error</option>
          <option value="warn">warn</option>
          <option value="info">info</option>
          <option value="debug">debug</option>
        </select>
        <label className="flex items-center gap-1 cursor-pointer select-none">
          <input type="checkbox" checked={jsonOnly} onChange={(e) => setJsonOnly(e.target.checked)} className="w-3 h-3 rounded-sm bg-surface border-outline-variant text-primary" />
          <span className="font-code-md text-[11px] text-on-surface-variant">json</span>
        </label>
        <div className="flex-1" />
        <button onClick={exportLogs} className="px-2 py-0.5 rounded font-code-md text-[11px] bg-[#1a1a1a] text-on-surface-variant hover:text-on-surface">
          Export
        </button>
      </div>
      <div
        ref={ref}
        onScroll={onScroll}
        className="flex-1 bg-[#0a0a0a] p-3 font-code-md text-[12px] text-[#d1d5db] overflow-y-auto sidebar-scroll rounded-b-lg min-h-[240px]"
      >
        {rows.length === 0 && <span className="text-on-surface-variant/50">Waiting for logs...</span>}
        {rows.length > 0 && filtered.length === 0 && (
          <span className="text-on-surface-variant/50">No lines match the current filter.</span>
        )}
        {filtered.map((row) => (
          <div key={row.id} className="whitespace-pre-wrap break-all py-[1px] hover:bg-white/[0.03] rounded px-1">
            <LogLine row={row} />
          </div>
        ))}
      </div>
    </div>
  );
}
