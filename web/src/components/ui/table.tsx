import React from "react";

export function Table({
  headers,
  children,
}: {
  headers: string[];
  children: React.ReactNode;
}) {
  return (
    <div className="overflow-x-auto rounded-xl border border-outline-variant/60 bg-surface-container-low/40">
      <table className="w-full border-collapse">
        <thead>
          <tr className="bg-surface-container-low/60">
            {headers.map((h) => (
              <th
                key={h}
                className="font-label-caps text-label-caps text-on-surface-variant uppercase text-left px-sm py-2.5 border-b border-outline-variant whitespace-nowrap"
              >
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-outline-variant/40">{children}</tbody>
      </table>
    </div>
  );
}
