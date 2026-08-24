import { useState, useEffect, useMemo } from "react";
import { getServer } from "../../../../api/client";
import { LogViewer, type LogLine as DesignLogLine } from "@aether/design-system";

const ANSI_RE = /\u001b\[[0-?]*[ -/]*[@-~]/g;
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

export function classify(line: string): LogRow {
  const text = line;
  const plainText = stripANSI(line);
  const json = tryPrettyJSON(plainText);
  const tsMatch = plainText.match(TIMESTAMP_RE);
  const tagMatch = plainText.match(TAG_RE);
  const lower = plainText.toLowerCase();
  let level = "";
  if (/\b(error|failed|falhou|fatal|panic|exception|traceback)\b/.test(lower)) level = "error";
  else if (/\b(warn|warning)\b/.test(lower)) level = "warn";
  else if (/\b(debug)\b/.test(lower)) level = "debug";
  else if (/\b(ok|ready|conclu|started|listening|running|healthy)\b/.test(lower)) level = "info";
  ROW_ID += 1;
  return { id: ROW_ID, text, json, ts: tsMatch ? tsMatch[0] : "", level, tag: tagMatch ? tagMatch[1].toLowerCase() : "" };
}

export function LiveLogs({ appId }: { appId: string }) {
  const [rows, setRows] = useState<LogRow[]>([]);
  const [follow, setFollow] = useState(true);

  useEffect(() => {
    const server = getServer();
    const es = new EventSource(server + "/api/v1/apps/" + appId + "/logs?follow=1", { withCredentials: true });
    es.onmessage = (ev) => {
      const chunks = ev.data.split("\n").filter((c: string) => c.trim() !== "");
      const newRows = chunks.map((c: string) => classify(c));
      setRows((prev) => [...prev.slice(-1000), ...newRows].slice(-1500));
    };
    return () => es.close();
  }, [appId]);

  const viewerLines = useMemo<DesignLogLine[]>(
    () => rows.map((row) => ({
      id: String(row.id),
      timestamp: row.ts || undefined,
      severity: row.level === "error" ? "error" : row.level === "warn" ? "warning" : row.level === "info" ? "info" : "info",
      message: row.json ?? (row.ts && row.text.startsWith(row.ts) ? row.text.slice(row.ts.length).trimStart() : row.text),
    })),
    [rows],
  );

  return (
    <LogViewer
      lines={viewerLines}
      followTail={follow}
      onFollowTailChange={setFollow}
    />
  );
}
