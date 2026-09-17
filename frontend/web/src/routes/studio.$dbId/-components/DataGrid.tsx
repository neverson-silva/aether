import type { StudioQueryResult } from "../../../api/types";

function fmtCell(v: unknown): string {
  if (v === null || v === undefined) return "NULL";
  if (typeof v === "object") return JSON.stringify(v);
  return String(v);
}

function cellClass(type: string | undefined, v: unknown): string {
  if (v === null || v === undefined) return "text-on-surface-variant/50 italic";
  const t = (type ?? "").toLowerCase();
  if (/(int|numeric|decimal|float|real|double|money|serial)/.test(t))
    return "text-tertiary text-right";
  if (/(bool)/.test(t)) return "text-secondary";
  if (/(date|time|timestamp)/.test(t)) return "text-status-warning";
  if (typeof v === "number") return "text-tertiary text-right";
  return "text-on-surface";
}

function shortType(t: string | undefined): string {
  if (!t) return "";
  const m = t.match(/^(\w+)(?:\(\d+(?:,\s*\d+)?\))?/);
  return (m ? m[1] : t).toLowerCase();
}

export function DataGrid({
  result,
  empty,
  types,
}: {
  result?: StudioQueryResult;
  empty?: string;
  types?: string[];
}) {
  if (!result) {
    return (
      <div className="flex items-center justify-center py-lg font-body-sm text-body-sm text-on-surface-variant">
        {empty ?? "Run a query or select a table to see data."}
      </div>
    );
  }

  const cols = result.columns ?? [];
  const rows = result.rows ?? [];

  return (
    <div className="flex h-full flex-col bg-surface-card">
      <div className="flex-1 overflow-auto sidebar-scroll">
        <table
          className="w-full border-collapse text-left"
          aria-label="Query results"
        >
          <caption className="sr-only">Query results</caption>
          <thead className="sticky top-0 z-10 border-b border-border bg-surface-container-low/95 shadow-sm backdrop-blur-xl">
            <tr>
              <th
                scope="col"
                className="w-12 border-r border-border px-sm py-1 text-center font-normal text-on-surface-variant"
              >
                #
              </th>
              {cols.map((c, i) => (
                <th
                  key={`${c}-${i}`}
                  scope="col"
                  className="whitespace-nowrap border-r border-border px-sm py-1 font-normal text-primary"
                >
                  <span className="flex items-center gap-1">
                    {c}
                    {types?.[i] && (
                      <span className="text-[10px] text-outline-variant font-normal">
                        {shortType(types[i])}
                      </span>
                    )}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {rows.map((row, ri) => (
              <tr
                key={`row-${ri}`}
                className="group transition-colors hover:bg-surface-container-high"
              >
                <td className="border-r border-border px-sm py-1 text-center text-outline-variant group-hover:text-on-surface-variant">
                  {ri + 1}
                </td>
                {row.map((cell, ci) => (
                  <td
                    key={ci}
                    className={`whitespace-nowrap border-r border-border px-sm py-1 ${cellClass(types?.[ci], cell)}`}
                  >
                    {fmtCell(cell)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {rows.length === 0 && (
        <div className="px-md py-lg font-body-sm text-body-sm text-on-surface-variant border-t border-outline-variant">
          No rows returned.
        </div>
      )}
    </div>
  );
}
