import { useMemo, useState } from "react";
import type { AppDetail } from "@/api/types";
import type { Deployment } from "@/api/types";
import { useCancelDeployment, useDeployCompare, useServiceCancelDeployment } from "@/hooks";
import { ArrowsClockwise, ArrowUUpLeft, X } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Button, Card, Checkbox, Dialog, EmptyState, Skeleton, useToast } from "@aether/design-system";
import { DeploymentLogModal } from "./DeploymentLogModal";

function cn(...classes: Array<string | false | undefined>) { return classes.filter(Boolean).join(" "); }
function isDeploymentActive(status: string) { return ["queued", "building", "starting", "health_checking"].includes(status); }
function deploymentTone(status: string) { return status === "failed" ? "danger" as const : status === "ready" || status === "running" ? "success" as const : isDeploymentActive(status) ? "warning" as const : "neutral" as const; }
function deploymentLabel(status: string) { return status === "ready" ? "success" : status; }

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

export function DeploymentsTab({ appId, serviceId, deployments, onRollback }: { appId: string; serviceId?: string; deployments: DeploymentRow[]; onRollback: () => void }) {
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [comparePair, setComparePair] = useState<{ a: string; b: string } | null>(null);
  const [logDep, setLogDep] = useState<string | null>(null);
  const [cancellingDep, setCancellingDep] = useState<string | null>(null);
  const { add } = useToast();
  const compare = useDeployCompare(serviceId ? "" : appId, serviceId ? null : comparePair?.a ?? null, serviceId ? null : comparePair?.b ?? null);
  const legacyCancelDeployment = useCancelDeployment(serviceId ? "" : appId);
  const serviceCancelDeployment = useServiceCancelDeployment(serviceId ?? "");
  const cancelDeployment = serviceId ? serviceCancelDeployment : legacyCancelDeployment;

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
      <Card variant="elevated" padding="none">
        <div className="flex flex-wrap items-start justify-between gap-md border-b border-outline-variant px-lg py-lg">
          <div>
            <p className="font-label-caps text-label-caps uppercase text-primary">Delivery history</p>
            <div className="mt-xs flex flex-wrap items-baseline gap-sm">
              <h2 className="font-headline-sm text-headline-sm text-on-surface">Deployments</h2>
              <span className="rounded-full bg-surface-container-high px-sm py-0.5 font-code-md text-code-md text-on-surface-variant">{ordered.length}</span>
            </div>
            <p className="mt-xs text-body-sm text-on-surface-variant">Review releases, compare changes, and inspect runtime output.</p>
          </div>
          <div className="flex items-center gap-sm">
            {!serviceId && sel.size === 2 && (
              <Button variant="secondary" icon={ArrowsClockwise as unknown as DesignIcon} onClick={() => { const [a, b] = [...sel]; setComparePair({ a, b }); }}>Compare</Button>
            )}
            {!serviceId && <Button variant="ghost" icon={ArrowUUpLeft as unknown as DesignIcon} onClick={onRollback}>Rollback</Button>}
          </div>
        </div>
        <div className="overflow-x-auto bg-surface-container-lowest/30">
          <table className="w-full min-w-[960px] border-collapse text-left">
            <thead>
              <tr className="border-b border-outline-variant bg-surface-container-low/60 font-label-caps text-label-caps uppercase text-on-surface-variant/60">
                <th scope="col" className="w-8 px-md py-sm" />
                <th scope="col" className="px-md py-sm">#</th>
                <th scope="col" className="px-md py-sm">Status</th>
                <th scope="col" className="px-md py-sm">Image</th>
                <th scope="col" className="px-md py-sm">Commit</th>
                <th scope="col" className="px-md py-sm">Duration</th>
                <th scope="col" className="px-md py-sm">Started</th>
                <th scope="col" className="px-md py-sm">Error</th>
                <th scope="col" className="px-md py-sm" />
              </tr>
            </thead>
            <tbody>
              {ordered.map((d) => {
                const fix = suggestFix(d.error);
                const dur = d.started_at && d.finished_at ? Math.round((new Date(d.finished_at).getTime() - new Date(d.started_at).getTime()) / 1000) : null;
                return (
                  <tr
                    key={d.id}
                    className={cn(
                      "border-b border-outline-variant/40 transition-[background-color,box-shadow] duration-200 hover:bg-surface-container-high",
                      isDeploymentActive(d.status) && "bg-status-success-container/10 shadow-[inset_3px_0_0_theme(colors.status.success)]"
                    )}
                  >
                    <td className="px-md py-md">
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
                    <td className="px-md py-md font-code-md text-code-md text-on-surface-variant">
                      <button
                        type="button"
                        className="rounded px-1 text-left outline-none hover:text-primary focus-visible:ring-2 focus-visible:ring-ring"
                        onClick={() => setLogDep(d.id)}
                        aria-label={`View logs for deployment #${d.number}`}
                      >
                        #{d.number}
                      </button>
                    </td>
                    <td className="px-md py-md">
                      <Badge tone={deploymentTone(d.status)}>{deploymentLabel(d.status)}</Badge>
                    </td>
                    <td className="max-w-[220px] truncate px-md py-md font-code-md text-code-md text-on-surface-variant">{d.image_ref || "—"}</td>
                    <td className="px-md py-md font-code-md text-code-md text-on-surface-variant/60">{(d.commit || "—").slice(0, 8)}</td>
                    <td className="px-md py-md font-code-md text-code-md text-on-surface-variant/60">{dur !== null ? `${dur}s` : "—"}</td>
                    <td className="px-md py-md font-code-md text-code-md text-on-surface-variant/60">{rowTime(d)}</td>
                    <td className="px-md py-md">
                      {d.error ? (
                        <span className="inline-block max-w-[180px] truncate rounded-md bg-error/10 px-sm py-0.5 align-middle font-code-md text-code-md text-error/80" title={d.error}>
                          {d.error}
                        </span>
                      ) : (
                        "—"
                      )}
                      {fix && (
                        <span className={`ml-xs px-1.5 py-0.5 rounded font-code-md text-code-md ${fix.level === "error" ? "bg-error/10 text-error" : "bg-status-warning-container/30 text-status-warning"}`} title={fix.detail}>
                          {fix.level === "error" ? "blocked" : "suggested"}
                        </span>
                      )}
                    </td>
                    <td className="px-md py-md text-right">
                      {isDeploymentActive(d.status) && (
                        <Button
                          variant="quiet"
                          size="sm"
                          icon={X as unknown as DesignIcon}
                          aria-label={`Cancel deployment #${d.number}`}
                          loading={cancelDeployment.isPending && cancellingDep === d.id}
                          onClick={(event) => {
                            event.stopPropagation();
                            setCancellingDep(d.id);
                            cancelDeployment.mutate(d.id, {
                              onError: (error) => add({ title: "Could not cancel deployment", description: error.message, tone: "error" }),
                              onSettled: () => setCancellingDep(null),
                            });
                          }}
                        >
                          Cancel
                        </Button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        {(deployments ?? []).length === 0 && <EmptyState title="No deployments yet" description="Deploy the service to create its first deployment." className="border-0" />}
      </Card>

      <Dialog open={!!comparePair} onOpenChange={(open) => { if (!open) setComparePair(null); }} title="Deploy comparison" trigger={<button type="button" className="hidden" aria-hidden="true" tabIndex={-1} />}>
        {compare.data ? (
          <div className="space-y-lg">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-md">
              <div className="p-sm rounded bg-surface-container-lowest border border-outline-variant">
                <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase mb-sm">Image</p>
                <p className="font-code-md text-code-md text-on-surface truncate">{compare.data.image.from || "—"}</p>
                <p className="font-code-md text-code-md text-status-success truncate">→ {compare.data.image.to || "—"}</p>
              </div>
              <div className="p-sm rounded bg-surface-container-lowest border border-outline-variant">
                <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase mb-sm">Commit</p>
                <p className="font-code-md text-code-md text-on-surface">{(compare.data.commit.from || "—").slice(0, 8)}</p>
                <p className="font-code-md text-code-md text-status-success">→ {(compare.data.commit.to || "—").slice(0, 8)}</p>
              </div>
            </div>
            {compare.data.env_added.length + compare.data.env_removed.length + compare.data.env_changed.length > 0 ? (
              <div className="space-y-md">
                {compare.data.env_added.length > 0 && (
                  <div>
                    <p className="font-label-caps text-label-caps text-status-success uppercase mb-sm">Added variables</p>
                    <div className="flex gap-sm flex-wrap">{compare.data.env_added.map((k) => <code key={k} className="px-2 py-0.5 rounded border border-status-success/30 font-code-md text-code-md text-status-success">{k}</code>)}</div>
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
                    <p className="font-label-caps text-label-caps text-status-warning uppercase mb-sm">Changed variables</p>
                    <div className="flex gap-sm flex-wrap">{compare.data.env_changed.map((k) => <code key={k} className="px-2 py-0.5 rounded border border-status-warning/30 font-code-md text-code-md text-status-warning">{k}</code>)}</div>
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

      <DeploymentLogModal appId={appId} serviceId={serviceId} deploymentId={logDep} onClose={() => setLogDep(null)} />
    </>
  );
}
