import { useState } from "react";
import { useAlertEvents, useAlertRules, useCreateAlertRule, useDeleteAlertRule, useResolveAlert, useApps } from "../../../../api/hooks";
import { Button, Card, Field, Input, Select, useToast } from "../../../../components/ui";

const METRIC_LABEL: Record<string, string> = { cpu: "CPU %", memory: "Memory (MiB)", memory_pct: "Memory %", disk: "Disk (MiB)" };
const METRIC_HINT: Record<string, string> = { cpu: "percent (0-100)", memory: "MiB", memory_pct: "percent (0-100)", disk: "MiB" };

export function AlertsPanel() {
  const { data: rules } = useAlertRules();
  const { data: events } = useAlertEvents();
  const { data: apps } = useApps();
  const createRule = useCreateAlertRule();
  const deleteRule = useDeleteAlertRule();
  const resolve = useResolveAlert();
  const { toast } = useToast();
  const [name, setName] = useState("");
  const [metric, setMetric] = useState("cpu");
  const [threshold, setThreshold] = useState("80");
  const [severity, setSeverity] = useState("warning");
  const [targetApp, setTargetApp] = useState("");

  const add = async () => {
    const t = parseFloat(threshold);
    if (!name.trim() || !isFinite(t)) {
      toast("Fill in name and a valid threshold", "error");
      return;
    }
    await createRule.mutateAsync({ name, metric, threshold: t, severity, target_app: targetApp, window_s: 30, enabled: true });
    toast(`Alert rule "${name}" created`);
    setName("");
    setThreshold("80");
  };

  return (
    <Card>
      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Alert rules</h2>
      <div className="grid grid-cols-2 md:grid-cols-5 gap-sm mb-sm">
        <Input icon="label" placeholder="Rule name" value={name} onChange={(e) => setName(e.target.value)} />
        <Select value={metric} onChange={(e) => setMetric(e.target.value)}>
          <option value="cpu">CPU %</option>
          <option value="memory">Memory (MiB)</option>
          <option value="memory_pct">Memory %</option>
        </Select>
        <Input icon="speed" placeholder={METRIC_HINT[metric]} value={threshold} onChange={(e) => setThreshold(e.target.value)} type="number" />
        <Select value={severity} onChange={(e) => setSeverity(e.target.value)}>
          <option value="warning">warning</option>
          <option value="critical">critical</option>
          <option value="info">info</option>
        </Select>
        <Select value={targetApp} onChange={(e) => setTargetApp(e.target.value)}>
          <option value="">All services</option>
          {(apps ?? []).map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </Select>
      </div>
      <Button onClick={add} className="mb-md">
        <span className="material-symbols-outlined text-[16px]">add</span>
        Add rule
      </Button>

      <div className="space-y-sm mb-lg">
        {(rules ?? []).map((r) => (
          <div key={r.id} className="flex items-center gap-sm p-sm rounded border border-outline-variant/60">
            <span className={`px-2 py-0.5 rounded font-code-md text-code-md ${r.severity === "critical" ? "bg-error/10 text-error" : r.severity === "warning" ? "bg-[#fbbf24]/10 text-[#fbbf24]" : "bg-primary/10 text-primary"}`}>
              {r.severity}
            </span>
            <span className="font-body-md text-body-md text-on-surface flex-1">{r.name}</span>
            <span className="font-code-md text-code-md text-on-surface-variant">{METRIC_LABEL[r.metric] ?? r.metric}</span>
            <span className="font-code-md text-code-md text-on-surface-variant">{"threshold: " + r.threshold}</span>
            <span className="font-code-md text-code-md text-on-surface-variant/60">{r.target_app ? "targeted" : "all services"}</span>
            <Button variant="ghost" onClick={() => deleteRule.mutateAsync(r.id)}>
              <span className="material-symbols-outlined text-[16px]">delete</span>
            </Button>
          </div>
        ))}
        {(rules ?? []).length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant">No alert rules yet. Add one above (e.g. CPU above 80%).</p>}
      </div>

      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Alert history</h2>
      <div className="space-y-sm max-h-[300px] overflow-y-auto sidebar-scroll">
        {(events ?? []).map((e) => (
          <div key={e.id} className={`flex items-start gap-sm p-sm rounded border ${e.resolved_at ? "border-outline-variant/40 opacity-60" : "border-error/30 bg-error/5"}`}>
            <span className={`material-symbols-outlined text-[16px] mt-0.5 ${e.resolved_at ? "text-on-surface-variant" : "text-error"}`}>
              {e.resolved_at ? "check_circle" : "error"}
            </span>
            <div className="flex-1 min-w-0">
              <p className="font-body-sm text-body-sm text-on-surface">{e.message}</p>
              <p className="font-code-md text-code-md text-on-surface-variant/60">
                {METRIC_LABEL[e.metric] ?? e.metric} {e.value.toFixed(1)} / {e.threshold} · {new Date(e.created_at).toLocaleString()}
                {e.resolved_at ? " · resolved" : ""}
              </p>
            </div>
            {!e.resolved_at && (
              <Button variant="ghost" onClick={() => resolve.mutateAsync(e.id)}>
                <span className="material-symbols-outlined text-[16px]">done</span>
              </Button>
            )}
          </div>
        ))}
        {(events ?? []).length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant">No alert events recorded.</p>}
      </div>
    </Card>
  );
}
