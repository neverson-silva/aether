import { useState } from "react";
import { AppWindow, Gear } from "@phosphor-icons/react";
import { Badge } from "@aether/design-system";
import type { MonitoringResource, ResourceOwner } from "../../../../hooks";
import { fmtBytes, fmtRate } from "./format";

function cn(...classes: Array<string | false | undefined>) { return classes.filter(Boolean).join(" "); }

const OWNER_BADGE: Record<ResourceOwner, string> = {
  aether: "bg-primary/10 text-primary border-primary/20",
  user: "bg-tertiary/10 text-tertiary border-tertiary/20",
  system: "bg-outline/10 text-outline border-outline/20",
  unknown: "bg-outline-variant/10 text-on-surface-variant border-outline-variant/30",
};

const OWNER_LABEL: Record<ResourceOwner, string> = {
  aether: "Aether",
  user: "User",
  system: "System",
  unknown: "Unknown",
};

export function ResourcesTable({
  resources,
  selectedId,
  onSelect,
}: {
  resources: MonitoringResource[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const [filter, setFilter] = useState<ResourceOwner | "all">("all");
  const rows = resources.filter((r) => filter === "all" || r.owner === filter);

  const count = (o: ResourceOwner) => resources.filter((r) => r.owner === o).length;

  return (
    <div className="bg-surface-container border border-outline-variant rounded-lg overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-sm p-sm border-b border-outline-variant bg-surface-container-high">
        <h2 className="font-label-caps text-label-caps text-on-surface">Resource Usage</h2>
        <div className="flex gap-xs">
          {(["all", "user", "aether", "unknown"] as const).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={cn(
                "px-2 py-1 rounded font-label-caps text-label-caps transition-colors border",
                filter === f ? "bg-primary/10 text-primary border-primary/30" : "border-outline-variant/50 text-on-surface-variant hover:text-on-surface",
              )}
            >
              {f === "all" ? "All" : OWNER_LABEL[f]}
              {f !== "all" ? ` (${count(f)})` : ""}
            </button>
          ))}
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse min-w-[760px]">
          <thead>
            <tr className="font-label-caps text-label-caps text-on-surface-variant/70 border-b border-outline-variant/50">
              <th className="py-2 pl-lg pr-2 font-normal">Resource</th>
              <th className="py-2 px-2 font-normal">Owner</th>
              <th className="py-2 px-2 font-normal">State</th>
              <th className="py-2 px-2 font-normal text-right">CPU</th>
              <th className="py-2 px-2 font-normal text-right">Memory</th>
              <th className="py-2 px-2 font-normal text-right">Storage</th>
              <th className="py-2 px-2 font-normal text-right">Network</th>
              <th className="py-2 pr-lg pl-2 font-normal text-right">Disk I/O</th>
            </tr>
          </thead>
          <tbody className="font-code-md text-code-md text-on-surface-variant divide-y divide-outline-variant/40">
            {rows.map((r) => {
              const selected = r.id === selectedId;
              return (
                <tr
                  key={r.id}
                  onClick={() => onSelect(r.id)}
                  className={cn(
                    "transition-colors cursor-pointer",
                    selected ? "bg-primary/5" : "table-row-hover",
                    !r.active && "opacity-50",
                  )}
                >
                  <td className="py-2 pl-lg pr-2 min-w-0">
                    <div className="flex items-center gap-2">
                      {r.owner === "user" ? <AppWindow size={16} className="shrink-0 text-muted-foreground" /> : <Gear size={16} className="shrink-0 text-muted-foreground" />}
                      <span className="text-on-surface truncate max-w-[220px]" title={r.id}>
                        {r.name}
                      </span>
                    </div>
                  </td>
                  <td className="py-2 px-2">
                    <Badge tone={r.owner === "aether" ? "info" : r.owner === "user" ? "accent" : "neutral"}>{OWNER_LABEL[r.owner]}</Badge>
                  </td>
                  <td className="py-2 px-2">
                    <div className="flex items-center gap-2">
                      <span className={cn("w-2 h-2 rounded-full inline-block", r.state === "running" ? "bg-status-success" : r.state === "dead" ? "bg-status-danger" : "bg-muted-foreground")} />
                      <span className="capitalize">{r.state}</span>
                    </div>
                  </td>
                  <td className="py-2 px-2 text-right text-on-surface">
                    {r.active ? (
                      <>
                        <span className="text-primary">{r.cpu_of_host.toFixed(1)}%</span>
                        <span className="text-on-surface-variant/50 text-[10px]"> / {r.cpu_percent.toFixed(0)}% core</span>
                      </>
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className="py-2 px-2 text-right text-on-surface">
                    {r.active ? <>{fmtBytes(r.mem_usage)}{r.mem_limit > 0 ? <span className="text-on-surface-variant/50"> / {fmtBytes(r.mem_limit)}</span> : null}</> : "—"}
                  </td>
                  <td className="py-2 px-2 text-right text-on-surface" title="Container writable layer + proportional share of mounted volumes">
                    {r.storage != null ? <span className="text-tertiary">{fmtBytes(r.storage)}</span> : "—"}
                  </td>
                  <td className="py-2 px-2 text-right text-on-surface">
                    {r.active && r.has_net_rate ? (
                      <>
                        <span className="text-[#4ade80]">↓{fmtRate(r.net_rx_rate)}</span>{" "}
                        <span className="text-on-surface-variant/60">↑{fmtRate(r.net_tx_rate)}</span>
                      </>
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className="py-2 pr-lg pl-2 text-right text-on-surface-variant/70">
                    {r.active && r.has_block_rate ? `${fmtRate(r.block_rx_rate)} · ${fmtRate(r.block_tx_rate)}` : "—"}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      {rows.length === 0 && (
        <p className="font-body-sm text-body-sm text-on-surface-variant/60 text-center py-lg">
          {filter === "all" ? "No user workloads are running." : `No ${OWNER_LABEL[filter]} resources.`}
        </p>
      )}
    </div>
  );
}
