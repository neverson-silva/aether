import { useState, useEffect, useMemo } from "react";
import { apiGet } from "@/api/client";
import {
  LogViewer,
  type LogLine as DesignLogLine,
} from "@aether/design-system";

const ANSI_RE = /\u001b\[[0-?]*[ -/]*[@-~]/g;
const TIMESTAMP_RE =
  /^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?/;
const TAG_RE = /^\[([a-z_\-]+)\]/i;
const MAX_LOG_ROWS = 1500;

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

type LogRow = {
  id: number;
  text: string;
  json: string | null;
  ts: string;
  level: string;
  tag: string;
};

let ROW_ID = 0;

export function classify(line: string): LogRow {
  const text = line;
  const plainText = stripANSI(line);
  const json = tryPrettyJSON(plainText);
  const tsMatch = plainText.match(TIMESTAMP_RE);
  const tagMatch = plainText.match(TAG_RE);
  const lower = plainText.toLowerCase();
  let level = "";
  if (/\b(error|failed|falhou|fatal|panic|exception|traceback)\b/.test(lower))
    level = "error";
  else if (/\b(warn|warning)\b/.test(lower)) level = "warn";
  else if (/\b(debug)\b/.test(lower)) level = "debug";
  else if (
    /\b(ok|ready|conclu|started|listening|running|healthy)\b/.test(lower)
  )
    level = "info";
  ROW_ID += 1;
  return {
    id: ROW_ID,
    text,
    json,
    ts: tsMatch ? tsMatch[0] : "",
    level,
    tag: tagMatch ? tagMatch[1].toLowerCase() : "",
  };
}

export function LiveLogs({
  serviceId,
  enabled = true,
  endpoint,
}: {
  serviceId: string;
  enabled?: boolean;
  endpoint?: string;
}) {
  const [rows, setRows] = useState<LogRow[]>([]);
  const [follow, setFollow] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled || !serviceId) return;
    const initialEndpoint =
      endpoint ?? "/api/v1/services/" + serviceId + "/logs";
    let cancelled = false;
    setLoading(true);
    setError(null);
    apiGet<{ logs: string }>(initialEndpoint)
      .then((result) => {
        if (cancelled) return;
        setRows(
          result.logs
            .split("\n")
            .filter((line) => line.trim() !== "")
            .slice(-MAX_LOG_ROWS)
            .map((line) => classify(line)),
        );
      })
      .catch(() => {
        if (!cancelled) {
          setRows([]);
          setError("Unable to load runtime logs.");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [serviceId, enabled, endpoint]);

  const viewerLines = useMemo<DesignLogLine[]>(
    () =>
      rows.map((row) => ({
        id: String(row.id),
        timestamp: row.ts || undefined,
        severity:
          row.level === "error"
            ? "error"
            : row.level === "warn"
              ? "warning"
              : row.level === "info"
                ? "info"
                : "info",
        message:
          row.json ??
          (row.ts && row.text.startsWith(row.ts)
            ? row.text.slice(row.ts.length).trimStart()
            : row.text),
      })),
    [rows],
  );

  if (error)
    return (
      <div
        role="alert"
        className="rounded-xl border border-error/40 bg-error/10 p-md font-body-sm text-body-sm text-error"
      >
        {error}
      </div>
    );
  if (!loading && rows.length === 0)
    return (
      <div className="rounded-xl border border-outline-variant bg-surface-container-low p-md font-body-sm text-body-sm text-on-surface-variant">
        No runtime logs are available.
      </div>
    );
  return (
    <LogViewer
      lines={viewerLines}
      followTail={follow}
      onFollowTailChange={setFollow}
    />
  );
}
