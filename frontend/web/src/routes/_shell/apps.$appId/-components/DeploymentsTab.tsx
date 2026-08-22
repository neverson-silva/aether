import { useMemo, useState } from "react";
import type { AppDetail } from "../../../../api/types";
import type { Deployment } from "../../../../api/types";
import { useDeployCompare } from "../../../../hooks";
import { ArrowsClockwise, ArrowUUpLeft, Check, GitBranch, WarningCircle } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Button, Card, Checkbox, Dialog, Skeleton, useToast } from "@aether/design-system";
import { DeploymentLogModal } from "./DeploymentLogModal";

function cn(...classes: Array<string | false | undefined>) { return classes.filter(Boolean).join(" "); }
function isDeploymentActive(status: string) { return ["queued", "building", "starting", "health_checking"].includes(status); }
function deploymentTone(status: string) { return status === "failed" ? "danger" as const : status === "ready" || status === "running" ? "success" as const : isDeploymentActive(status) ? "warning" as const : "neutral" as const; }

export interface DeploymentRow extends Deployment {
  created_at: string;
  number: number;
}

export function suggestFix(error: string): { level: "error" | "warning"; title: string; detail: string } | null {
  if (!error) return null;
  const e = error.toLowerCase();
  const checks: { m: RegExp; level: "error" | "warning"; title: string; detail: string }[] = [
    { m: /out of memory|oom|memory.*limit|cannot allocate memory/i, level: "error", title: "Out of memory", detail: "The container exceeded its memory limit. Increase the memory allocation in Settings → Resources, or reduce the app's memory footprint." },
    { m: /exit code 0|exit code 1|process exited/i, level: "warning", title: "Process exited", detail: "The container's main process exited unexpectedly. Check the logs for the app's startup error and verify the start command." },
    { m: /port.*(?:already|in use)|address already in use|bind.*failed/i, level: "error", title: "Port conflict", detail: "Another container is already using the app port. Change the port in Settings → Network or stop the conflicting container." },
    { m: /no such image|manifest unknown|not found.*image|pull access denied/i, level: "error", title: "Image unavailable", detail: "The image could not be pulled. Check the image name/tag and that the registry credentials are correct in Settings → Registry." },
    { m: /unauthorized|authentication required|denied.*access/i, level: "error", title: "Registry authentication", detail: "The registry rejected the credentials. Update the registry settings for this app." },
    { m: /dockerfile.*not found|dockerfile.*missing|no dockerfile/i, level: "warning", title: "Dockerfile missing", detail: "No Dockerfile was found in the repository. Switch the build method to SmartBuild (CNB), or add a Dockerfile to the repo." },
    { m: /health.?check.*fail|timeout.*health|container is unhealthy|port.*not.*open/i, level: "warning", title: "Health check timeout", detail: "The container started but the health check did not pass. Verify the app listens on the configured port and path." },
    { m: /connection refused|network is unreachable|name or service not known/i, level: "error", title: "Connection failure", detail: "The app could not reach a dependency. Check environment variables (database URLs) and service discovery names." },
    { m: /cannot find module|module not found|no such file/i, level: "warning", title: "Build error", detail: "The build failed due to a missing module or file. Review the logs around the error and check the build command." },
    { m: /failed to parse|syntax error/i, level: "error", title: "Configuration error", detail: "The configuration (compose/build) could not be parsed. Validate the file syntax and retry." },
    { m: /timeout|timed out/i, level: "warning", title: "Timed out", detail: "An operation exceeded its time limit. Increase the deployment timeout or check network latency." },
    { m: /no such host|dns/i, level: "warning", title: "DNS resolution", detail: "A hostname could not be resolved. Check the domain configuration in Settings → Domains." },
  ];
  for (const c of checks) {
    if (c.m.test(e)) return { level: c.level, title: c.title, detail: c.detail };
  }
  return null;
}

export function DeploymentsTab({ appId, deployments, onRollback }: { appId: string; deployments: DeploymentRow[]; onRollback: () => void }) {
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [comparePair, setComparePair] = useState<{ a: string; b: string } | null>(null);
  const [logDep, setLogDep] = useState<string | null>(null);
  const { add } = useToast();
  const compare = useDeployCompare(appId, comparePair?.a ?? null, comparePair?.b ?? null);

  const toggle = (id: string) => {
    setSel((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else {
        if (next.size >= 2) {
          add({ title: "Select exactly two deployments to compare", tone: "info" });
          return prev;
        }
        next.add(id);
      }
      return next;
    });
  };

  const rowTime = (d: DeploymentRow) => new Date(d.created_at).toLocaleString(undefined, { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });

  const ordered = useMemo(() => [...(deployments ?? [])].sort((a, b) => b.number - a.number), [deployments]);

  return (
    <>
      <Card>
        <div className="flex items-center justify-between mb-md">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Deployments</h2>
          <div className="flex items-center gap-sm">
            {sel.size === 2 && (
              <Button variant="secondary" icon={ArrowsClockwise as unknown as DesignIcon} onClick={() => { const [a, b] = [...sel]; setComparePair({ a, b }); }}>Compare</Button>
            )}
            <Button variant="ghost" icon={ArrowUUpLeft as unknown as DesignIcon} onClick={onRollback}>Rollback</Button>
          </div>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-outline-variant font-label-caps text-label-caps text-on-surface-variant/60 uppercase">
                <th className="px-sm py-2 w-8" />
                <th className="px-sm py-2">#</th>
                <th className="px-sm py-2">Status</th>
                <th className="px-sm py-2">Image</th>
                <th className="px-sm py-2">Commit</th>
                <th className="px-sm py-2">Duration</th>
                <th className="px-sm py-2">Started</th>
                <th className="px-sm py-2">Error</th>
              </tr>
            </thead>
            <tbody>
              {ordered.map((d) => {
                const fix = suggestFix(d.error);
                const dur = d.started_at && d.finished_at ? Math.round((new Date(d.finished_at).getTime() - new Date(d.started_at).getTime()) / 1000) : null;
                return (
                  <tr
                    key={d.id}
                    onClick={() => setLogDep(d.id)}
                    className={cn(
                      "hover:bg-surface-container-high transition-colors border-b border-outline-variant/40 cursor-pointer",
                      isDeploymentActive(d.status) && "rt-card-active bg-[#4ade80]/[0.03]"
                    )}
                  >
                    <td className="px-sm py-2">
                      <input
                        type="checkbox"
                        checked={sel.has(d.id)}
                        onChange={(e) => {
                          e.stopPropagation();
                          toggle(d.id);
                        }}
                        className="size-4 rounded-sm bg-surface border-outline-variant text-primary"
                      />
                    </td>
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">#{d.number}</td>
                    <td className="px-sm py-2">
                      <Badge tone={deploymentTone(d.status)}>{d.status}</Badge>
                    </td>
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant max-w-[220px] truncate">{d.image_ref || "—"}</td>
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">{(d.commit || "—").slice(0, 8)}</td>
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">{dur !== null ? `${dur}s` : "—"}</td>
                    <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">{rowTime(d)}</td>
                    <td className="px-sm py-2">
                      {d.error ? (
                        <span className="font-code-md text-code-md text-error/80 max-w-[180px] truncate inline-block align-middle" title={d.error}>
                          {d.error}
                        </span>
                      ) : (
                        "—"
                      )}
                      {fix && (
                        <span className={`ml-xs px-1.5 py-0.5 rounded font-code-md text-code-md ${fix.level === "error" ? "bg-error/10 text-error" : "bg-[#fbbf24]/10 text-[#fbbf24]"}`} title={fix.detail}>
                          {fix.level === "error" ? "blocked" : "suggested"}
                        </span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        {(deployments ?? []).length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">No deployments yet.</p>}
      </Card>

      <Dialog open={!!comparePair} onOpenChange={(open) => { if (!open) setComparePair(null); }} title="Deploy comparison" trigger={<button type="button" className="hidden" aria-hidden="true" tabIndex={-1} />}>
        {compare.data ? (
          <div className="space-y-lg">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-md">
              <div className="p-sm rounded bg-surface-container-lowest border border-outline-variant">
                <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase mb-sm">Image</p>
                <p className="font-code-md text-code-md text-on-surface truncate">{compare.data.image.from || "—"}</p>
                <p className="font-code-md text-code-md text-[#4ade80] truncate">→ {compare.data.image.to || "—"}</p>
              </div>
              <div className="p-sm rounded bg-surface-container-lowest border border-outline-variant">
                <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase mb-sm">Commit</p>
                <p className="font-code-md text-code-md text-on-surface">{(compare.data.commit.from || "—").slice(0, 8)}</p>
                <p className="font-code-md text-code-md text-[#4ade80]">→ {(compare.data.commit.to || "—").slice(0, 8)}</p>
              </div>
            </div>
            {compare.data.env_added.length + compare.data.env_removed.length + compare.data.env_changed.length > 0 ? (
              <div className="space-y-md">
                {compare.data.env_added.length > 0 && (
                  <div>
                    <p className="font-label-caps text-label-caps text-[#4ade80] uppercase mb-sm">Added variables</p>
                    <div className="flex gap-sm flex-wrap">{compare.data.env_added.map((k) => <code key={k} className="px-2 py-0.5 rounded border border-[#4ade80]/30 font-code-md text-code-md text-[#4ade80]">{k}</code>)}</div>
                  </div>
                )}
                {compare.data.env_removed.length > 0 && (
                  <div>
                    <p className="font-label-caps text-label-caps text-error uppercase mb-sm">Removed variables</p>
                    <div className="flex gap-sm flex-wrap">{compare.data.env_removed.map((k) => <code key={k} className="px-2 py-0.5 rounded border border-error/30 font-code-md text-code-md text-error">{k}</code>)}</div>
                  </div>
                )}
                {compare.data.env_changed.length > 0 && (
                  <div>
                    <p className="font-label-caps text-label-caps text-[#fbbf24] uppercase mb-sm">Changed variables</p>
                    <div className="flex gap-sm flex-wrap">{compare.data.env_changed.map((k) => <code key={k} className="px-2 py-0.5 rounded border border-[#fbbf24]/30 font-code-md text-code-md text-[#fbbf24]">{k}</code>)}</div>
                  </div>
                )}
              </div>
            ) : (
              <p className="font-body-sm text-body-sm text-on-surface-variant">No environment variable differences between these deployments.</p>
            )}
          </div>
        ) : (
          <div className="space-y-sm" aria-label="Loading deployment comparison"><Skeleton variant="text" /><Skeleton variant="text" /><Skeleton variant="text" /></div>
        )}
      </Dialog>

      <DeploymentLogModal appId={appId} deploymentId={logDep} onClose={() => setLogDep(null)} />
    </>
  );
}
