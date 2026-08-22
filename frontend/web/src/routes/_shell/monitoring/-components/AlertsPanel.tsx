import { useState } from "react";
import { Check, Plus, Tag, Trash, XCircle } from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import { Badge, Button, Card, Input, NativeSelect, useToast } from "@aether/design-system";
import { useAlertEvents, useAlertRules, useCreateAlertRule, useDeleteAlertRule, useResolveAlert, useApps } from "../../../../hooks";

const METRIC_LABEL: Record<string, string> = { cpu: "CPU %", memory: "Memory (MiB)", memory_pct: "Memory %", disk: "Disk (MiB)" };
const METRIC_HINT: Record<string, string> = { cpu: "percent (0-100)", memory: "MiB", memory_pct: "percent (0-100)", disk: "MiB" };

export function AlertsPanel() {
  const { data: rules } = useAlertRules();
  const { data: events } = useAlertEvents();
  const { data: apps } = useApps();
  const createRule = useCreateAlertRule();
  const deleteRule = useDeleteAlertRule();
  const resolve = useResolveAlert();
  const { add } = useToast();
  const [name, setName] = useState("");
  const [metric, setMetric] = useState("cpu");
  const [threshold, setThreshold] = useState("80");
  const [severity, setSeverity] = useState("warning");
  const [targetApp, setTargetApp] = useState("");

  const addRule = async () => {
    const t = parseFloat(threshold);
    if (!name.trim() || !isFinite(t)) {
      add({ title: "Invalid rule", description: "Fill in a name and a valid threshold.", tone: "error" });
      return;
    }
    await createRule.mutateAsync({ name, metric, threshold: t, severity, target_app: targetApp, window_s: 30, enabled: true });
    setName("");
    setThreshold("80");
  };

  return (
    <Card>
      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Alert rules</h2>
      <div className="grid grid-cols-2 md:grid-cols-5 gap-sm mb-sm">
        <Input leadingIcon={Tag as unknown as DesignIcon} placeholder="Rule name" value={name} onChange={(e) => setName(e.target.value)} />
        <NativeSelect value={metric} onChange={(e) => setMetric(e.target.value)} options={[{ label: "CPU %", value: "cpu" }, { label: "Memory (MiB)", value: "memory" }, { label: "Memory %", value: "memory_pct" }]} />
        <Input placeholder={METRIC_HINT[metric]} value={threshold} onChange={(e) => setThreshold(e.target.value)} type="number" />
        <NativeSelect value={severity} onChange={(e) => setSeverity(e.target.value)} options={[{ label: "warning", value: "warning" }, { label: "critical", value: "critical" }, { label: "info", value: "info" }]} />
        <NativeSelect value={targetApp} onChange={(e) => setTargetApp(e.target.value)} options={[{ label: "All services", value: "" }, ...(apps ?? []).map((a) => ({ label: a.name, value: a.id }))]} />
      </div>
      <Button onClick={addRule} className="mb-md">
        <Plus size={16} />Add rule
      </Button>

      <div className="space-y-sm mb-lg">
        {(rules ?? []).map((r) => (
          <div key={r.id} className="flex items-center gap-sm p-sm rounded border border-outline-variant/60">
            <Badge tone={r.severity === "critical" ? "danger" : r.severity === "warning" ? "warning" : "info"}>{r.severity}</Badge>
            <span className="font-body-md text-body-md text-on-surface flex-1">{r.name}</span>
            <span className="font-code-md text-code-md text-on-surface-variant">{METRIC_LABEL[r.metric] ?? r.metric}</span>
            <span className="font-code-md text-code-md text-on-surface-variant">{"threshold: " + r.threshold}</span>
            <span className="font-code-md text-code-md text-on-surface-variant/60">{r.target_app ? "targeted" : "all services"}</span>
            <Button variant="ghost" onClick={() => deleteRule.mutateAsync(r.id)}>
              <Trash size={16} />
            </Button>
          </div>
        ))}
        {(rules ?? []).length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant">No alert rules yet. Add one above (e.g. CPU above 80%).</p>}
      </div>

      <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">Alert history</h2>
      <div className="space-y-sm max-h-[300px] overflow-y-auto sidebar-scroll">
        {(events ?? []).map((e) => (
          <div key={e.id} className={`flex items-start gap-sm p-sm rounded border ${e.resolved_at ? "border-outline-variant/40 opacity-60" : "border-error/30 bg-error/5"}`}>
            {e.resolved_at ? <Check size={16} className="mt-0.5 text-muted-foreground" /> : <XCircle size={16} className="mt-0.5 text-status-danger" />}
            <div className="flex-1 min-w-0">
              <p className="font-body-sm text-body-sm text-on-surface">{e.message}</p>
              <p className="font-code-md text-code-md text-on-surface-variant/60">
                {METRIC_LABEL[e.metric] ?? e.metric} {e.value.toFixed(1)} / {e.threshold} · {new Date(e.created_at).toLocaleString()}
                {e.resolved_at ? " · resolved" : ""}
              </p>
            </div>
            {!e.resolved_at && (
              <Button variant="ghost" onClick={() => resolve.mutateAsync(e.id)}>
                <Check size={16} />
              </Button>
            )}
          </div>
        ))}
        {(events ?? []).length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant">No alert events recorded.</p>}
      </div>
    </Card>
  );
}
