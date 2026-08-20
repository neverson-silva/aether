import type { StudioQueryResult } from "../../../../../api/types";

function fmtCell(v: unknown): string {
  if (v === null || v === undefined) return "NULL";
  if (typeof v === "object") return JSON.stringify(v);
  return String(v);
}

function cellClass(type: string | undefined, v: unknown): string {
  if (v === null || v === undefined) return "text-on-surface-variant/50 italic";
  const t = (type ?? "").toLowerCase();
  if (/(int|numeric|decimal|float|real|double|money|serial)/.test(t)) return "text-[#f78c6c] text-right";
  if (/(bool)/.test(t)) return "text-[#c792ea]";
  if (/(date|time|timestamp)/.test(t)) return "text-[#ecc48d]";
  if (typeof v === "number") return "text-[#f78c6c] text-right";
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
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-auto sidebar-scroll">
        <table className="w-full text-left border-collapse">
          <thead className="sticky top-0 bg-surface-container-low border-b border-outline-variant shadow-sm z-10">
            <tr>
              <th className="py-1 px-sm font-normal text-on-surface-variant border-r border-outline-variant w-12 text-center">#</th>
              {cols.map((c, i) => (
                <th key={i} className="py-1 px-sm font-normal text-[#82aaff] border-r border-outline-variant whitespace-nowrap">
                  <span className="flex items-center gap-1">
                    {c}
                    {types?.[i] && <span className="text-[10px] text-outline-variant font-normal">{shortType(types[i])}</span>}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-outline-variant">
            {rows.map((row, ri) => (
              <tr key={ri} className="hover:bg-[#161616] group">
                <td className="py-1 px-sm text-outline-variant text-center border-r border-outline-variant group-hover:text-on-surface-variant">{ri + 1}</td>
                {row.map((cell, ci) => (
                  <td key={ci} className={`py-1 px-sm border-r border-outline-variant whitespace-nowrap ${cellClass(types?.[ci], cell)}`}>
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