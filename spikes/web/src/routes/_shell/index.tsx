import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useExportOrg, useSystemSummary } from "../../api/hooks";
import { useToast } from "../../components/ui";

function fmtBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MiB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GiB`;
}

function statusStyle(status: string): { dot: string; label: string } {
  switch (status) {
    case "healthy":
      return { dot: "bg-primary", label: "text-primary" };
    case "syncing":
      return { dot: "bg-tertiary-fixed-dim", label: "text-tertiary-fixed-dim" };
    case "degraded":
      return { dot: "bg-error animate-pulse", label: "text-error" };
    default:
      return { dot: "bg-on-surface-variant/50", label: "text-on-surface-variant" };
  }
}

function Dashboard() {
  const navigate = useNavigate();
  const { data: summary } = useSystemSummary();
  const exportOrg = useExportOrg();
  const { toast } = useToast();

  const doExport = async () => {
    try {
      const yaml = await exportOrg.mutateAsync();
      const blob = new Blob([yaml], { type: "application/yaml" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "aether.yml";
      a.click();
      URL.revokeObjectURL(url);
      toast("Exported aether.yml");
    } catch (err) {
      toast(err instanceof Error ? err.message : "export failed", "error");
    }
  };

  const health = summary?.health_pct ?? 0;
  const traffic = fmtBytes(summary?.traffic_bytes ?? 0);
  const deployments = summary?.deployments ?? 0;

  return (
    <div className="space-y-lg">
      <div className="flex flex-wrap justify-between items-end gap-md mb-xl">
        <div className="min-w-0">
          <h2 className="font-display-lg text-[clamp(1.75rem,4vw,3rem)] leading-[1.1] text-on-surface mb-2">Fleet Overview</h2>
          <p className="font-code-md text-code-md text-on-surface-variant flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-primary animate-pulse" />
            System Operational • Last updated: Just now
          </p>
        </div>
        <div className="flex gap-4">
          <button
            onClick={doExport}
            className="flex items-center gap-2 px-4 py-2 border border-outline-variant rounded font-label-caps text-label-caps text-on-surface hover:border-primary hover:text-primary transition-all bg-surface"
          >
            <span className="material-symbols-outlined text-[18px]">download</span>
            Export Report
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-lg">
        <div className="glass-panel rounded-xl p-lg relative overflow-hidden group hover:border-primary/50 transition-colors">
          <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <span className="material-symbols-outlined text-[64px] text-primary">health_and_safety</span>
          </div>
          <p className="font-label-caps text-label-caps text-on-surface-variant mb-2">Global Health</p>
          <div className="flex items-baseline gap-2">
            <span className="font-display-lg text-display-lg text-on-surface">{health.toFixed(1)}</span>
            <span className="font-headline-sm text-headline-sm text-primary">%</span>
          </div>
          <div className="mt-4 flex items-center gap-2 text-primary font-code-md text-code-md">
            <span className="material-symbols-outlined text-[16px]">arrow_upward</span>
            <span>from live probes</span>
          </div>
        </div>

        <div className="glass-panel rounded-xl p-lg relative overflow-hidden group hover:border-secondary/50 transition-colors">
          <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <span className="material-symbols-outlined text-[64px] text-secondary">swap_vert</span>
          </div>
          <p className="font-label-caps text-label-caps text-on-surface-variant mb-2">Total Traffic</p>
          <div className="flex items-baseline gap-2">
            <span className="font-display-lg text-display-lg text-on-surface">{traffic}</span>
            <span className="font-headline-sm text-headline-sm text-secondary">net</span>
          </div>
          <div className="mt-4 flex items-center gap-2 text-on-surface-variant font-code-md text-code-md">
            <span className="material-symbols-outlined text-[16px]">horizontal_rule</span>
            <span>since container start</span>
          </div>
        </div>

        <div className="glass-panel rounded-xl p-lg relative overflow-hidden group hover:border-tertiary-fixed-dim/50 transition-colors">
          <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <span className="material-symbols-outlined text-[64px] text-tertiary-fixed-dim">rocket_launch</span>
          </div>
          <p className="font-label-caps text-label-caps text-on-surface-variant mb-2">Total Deployments</p>
          <div className="flex items-baseline gap-2">
            <span className="font-display-lg text-display-lg text-on-surface">{deployments}</span>
            <span className="font-headline-sm text-headline-sm text-tertiary-fixed-dim">runs</span>
          </div>
          <div className="mt-4 flex items-center gap-2 text-on-surface-variant font-code-md text-code-md">
            <span className="material-symbols-outlined text-[16px]">horizontal_rule</span>
            <span>across all services</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-lg">
        <div className="lg:col-span-2 glass-panel rounded-xl flex flex-col">
          <div className="p-lg border-b border-outline-variant/30 flex justify-between items-center">
            <h3 className="font-headline-sm text-headline-sm text-on-surface">Active Projects</h3>
            <Link to="/projects" className="text-primary font-label-caps text-label-caps hover:underline">
              View All
            </Link>
          </div>
          <div className="p-0 overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="font-label-caps text-label-caps text-on-surface-variant border-b border-outline-variant/30">
                  <th className="p-md pl-lg font-normal">Project Name</th>
                  <th className="p-md font-normal">Environment</th>
                  <th className="p-md font-normal">Status</th>
                  <th className="p-md font-normal text-right pr-lg">Last Deploy</th>
                </tr>
              </thead>
              <tbody className="font-body-md text-body-md text-on-surface divide-y divide-outline-variant/20">
                {(summary?.projects ?? []).length === 0 && (
                  <tr>
                    <td colSpan={4} className="p-lg font-body-sm text-body-sm text-on-surface-variant">
                      No projects yet. Create one to get started.
                    </td>
                  </tr>
                )}
                {(summary?.projects ?? []).map((p, i) => {
                  const st = statusStyle(p.status);
                  return (
                    <tr
                      key={p.id}
                      onClick={() => navigate({ to: "/projects/$projectId", params: { projectId: p.id } })}
                      className="hover:bg-surface-container-high/50 transition-colors group cursor-pointer"
                    >
                      <td className="p-md pl-lg font-code-md">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded bg-surface-container flex items-center justify-center border border-outline-variant group-hover:border-primary transition-colors">
                            <span className="material-symbols-outlined text-[18px] text-primary">public</span>
                          </div>
                          {p.name}
                        </div>
                      </td>
                      <td className="p-md">
                        <span className="px-2 py-1 rounded bg-surface-container-high border border-outline-variant text-xs text-on-surface-variant">
                          {p.env}
                        </span>
                      </td>
                      <td className="p-md">
                        <div className="flex items-center gap-2">
                          <span className={`w-2 h-2 rounded-full ${st.dot}`} />
                          <span className={st.label}>{p.status}</span>
                        </div>
                      </td>
                      <td className="p-md text-right pr-lg font-code-md text-on-surface-variant text-xs">{p.last_deploy}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        <div className="glass-panel rounded-xl flex flex-col p-lg">
          <h3 className="font-headline-sm text-headline-sm text-on-surface mb-lg">Resource Usage</h3>
          <div className="space-y-6 flex-1 flex flex-col justify-center">
            <div>
              <div className="flex justify-between items-end mb-2">
                <span className="font-label-caps text-label-caps text-on-surface-variant">CPU Allocation</span>
                <span className="font-code-md text-code-md text-on-surface">{Math.round(summary?.cpu_pct ?? 0)}%</span>
              </div>
              <div className="h-2 w-full bg-surface-container-high rounded-full overflow-hidden border border-outline-variant/30">
                <div
                  className="h-full bg-primary rounded-full transition-all duration-1000 ease-out"
                  style={{ width: `${Math.min(100, summary?.cpu_pct ?? 0)}%` }}
                />
              </div>
            </div>
            <div>
              <div className="flex justify-between items-end mb-2">
                <span className="font-label-caps text-label-caps text-on-surface-variant">Memory (RAM)</span>
                <span className="font-code-md text-code-md text-on-surface">{Math.round(summary?.mem_pct ?? 0)}%</span>
              </div>
              <div className="h-2 w-full bg-surface-container-high rounded-full overflow-hidden border border-outline-variant/30">
                <div
                  className="h-full bg-tertiary-fixed-dim rounded-full transition-all duration-1000 ease-out"
                  style={{ width: `${Math.min(100, summary?.mem_pct ?? 0)}%` }}
                />
              </div>
            </div>
            <div>
              <div className="flex justify-between items-end mb-2">
                <span className="font-label-caps text-label-caps text-on-surface-variant">Storage I/O</span>
                <span className="font-code-md text-code-md text-on-surface">{Math.round(summary?.io_pct ?? 0)}%</span>
              </div>
              <div className="h-2 w-full bg-surface-container-high rounded-full overflow-hidden border border-outline-variant/30">
                <div
                  className="h-full bg-secondary rounded-full transition-all duration-1000 ease-out"
                  style={{ width: `${Math.min(100, summary?.io_pct ?? 0)}%` }}
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export const Route = createFileRoute("/_shell/")({
  component: Dashboard,
});
