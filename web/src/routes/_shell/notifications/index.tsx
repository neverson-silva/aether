import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  useAlertEvents,
  useAlertRules,
  useCreateAlertRule,
  useNotificationChannels,
  useOutWebhooks,
  useSetAlertRuleEnabled,
} from "../../../hooks";
import type { AlertRule } from "../../../hooks";
import { useApps } from "../../../hooks";
import { useNotifications as useProviderNotifications } from "../../../components/NotificationProvider";
import { Button, Field, Input, Modal, Select, useToast } from "../../../components/ui";
import { AppButton } from "../../../components/ds";

const METRIC_META: Record<string, { label: string; icon: string }> = {
  cpu: { label: "CPU", icon: "memory" },
  memory: { label: "Memory", icon: "memory" },
  memory_pct: { label: "Memory %", icon: "memory" },
  disk: { label: "Disk", icon: "storage" },
};

function severityBadge(severity: string): { cls: string; label: string } {
  switch (severity) {
    case "critical":
      return { cls: "bg-error/10 text-error border-error/20", label: "CRITICAL" };
    case "warning":
      return { cls: "bg-tertiary/10 text-tertiary border-tertiary/20", label: "WARNING" };
    default:
      return { cls: "bg-surface-container-highest text-outline border-outline-variant", label: "INFO" };
  }
}

function severityIconBg(severity: string): string {
  switch (severity) {
    case "critical":
      return "bg-error-container";
    case "warning":
      return "bg-tertiary-container/20";
    default:
      return "bg-surface-container-highest";
  }
}

function timeAgo(iso: string): string {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

function notifMeta(type: string): { icon: string; color: string } {
  if (type.includes("failed")) return { icon: "error", color: "text-error" };
  if (type.includes("ready") || type.includes("finished")) return { icon: "check_circle", color: "text-[#4ade80]" };
  if (type.includes("queued") || type.includes("building") || type.includes("starting") || type.includes("healthcheck")) return { icon: "hourglass_top", color: "text-[#fbbf24]" };
  if (type.includes("server")) return { icon: "dns", color: "text-[#60a5fa]" };
  if (type.includes("backup")) return { icon: "backup", color: "text-[#4ade80]" };
  if (type.includes("alert")) return { icon: "notifications_active", color: "text-error" };
  return { icon: "notifications", color: "text-on-surface-variant" };
}

function ChannelIcon({ type }: { type: string }) {
  const map: Record<string, { icon: string; color: string }> = {
    slack: { icon: "forum", color: "text-[#E51475]" },
    discord: { icon: "chat_bubble", color: "text-[#5865F2]" },
    telegram: { icon: "send", color: "text-[#229ED9]" },
    email: { icon: "mail", color: "text-outline" },
    webhook: { icon: "webhook", color: "text-outline" },
  };
  const m = map[type] ?? { icon: "hub", color: "text-outline" };
  return <span className={`material-symbols-outlined ${m.color}`} style={{ fontVariationSettings: "'FILL' 1" }}>{m.icon}</span>;
}

function Notifications() {
  const { data: rules } = useAlertRules();
  const { data: channels } = useNotificationChannels();
  const { data: hooks } = useOutWebhooks();
  const { data: events } = useAlertEvents(20);
  const { data: apps } = useApps();
  const createRule = useCreateAlertRule();
  const setEnabled = useSetAlertRuleEnabled();
  const { toast } = useToast();
  const navigate = useNavigate();
  const { list: notifList, unread, markRead, markAllRead } = useProviderNotifications();

  const [tab, setTab] = useState<"notifications" | "alerts">("notifications");
  const [showOnlyUnread, setShowOnlyUnread] = useState(false);
  const [filter, setFilter] = useState("");
  const [ruleOpen, setRuleOpen] = useState(false);
  const [name, setName] = useState("");
  const [metric, setMetric] = useState("cpu");
  const [threshold, setThreshold] = useState("80");
  const [severity, setSeverity] = useState("warning");
  const [targetApp, setTargetApp] = useState("");

  const filteredRules = (rules ?? []).filter((r) =>
    !filter || (r.name.toLowerCase() + " " + r.metric).includes(filter.toLowerCase())
  );

  const addRule = async () => {
    const t = parseFloat(threshold);
    if (!name.trim() || !isFinite(t)) {
      toast("Fill in a name and a valid threshold", "error");
      return;
    }
    try {
      await createRule.mutateAsync({ name, metric, threshold: t, severity, target_app: targetApp, window_s: 30, enabled: true });
      setRuleOpen(false);
      setName("");
      setThreshold("80");
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create rule", "error");
    }
  };

  const channelStatus = (type: string, enabled?: boolean): { text: string; connected: boolean } => {
    const active = channels?.filter((c) => c.type === type && c.enabled).length ?? 0;
    if (active > 0) return { text: "CONNECTED", connected: true };
    if ((channels ?? []).some((c) => c.type === type)) return { text: "CONFIGURED", connected: false };
    if (type === "webhook" && (hooks ?? []).length > 0) return { text: `${hooks!.length} ACTIVE`, connected: true };
    return { text: "NOT SETUP", connected: false };
  };

  const toggleRule = (rule: AlertRule) => {
    setEnabled.mutate({ id: rule.id, enabled: !rule.enabled });
    toast(rule.enabled ? `Rule "${rule.name}" disabled` : `Rule "${rule.name}" enabled`);
  };

  return (
    <div className="max-w-[1600px] mx-auto">
      <div className="mb-lg flex flex-col md:flex-row justify-between items-start md:items-end gap-md">
        <div>
          <h1 className="font-headline-sm text-headline-sm font-bold text-on-surface tracking-tight mb-1">Notifications &amp; Alerts</h1>
          <p className="font-body-sm text-body-sm text-on-surface-variant">Notifications history, alert rules, routing and communication channels.</p>
        </div>
        <div className="flex gap-sm">
          <Button variant="subtle" leftIcon="history" onClick={() => navigate({ to: "/monitoring" } as never)}>
            View Audit Log
          </Button>
          <AppButton leftIcon="add" onClick={() => setRuleOpen(true)}>
            New Rule
          </AppButton>

      <div className="flex items-center gap-sm border-b border-outline-variant mb-lg">
        <button
          onClick={() => setTab("notifications")}
          className={`flex items-center gap-sm px-md py-2.5 font-label-caps text-label-caps uppercase border-b-2 -mb-px transition-colors ${
            tab === "notifications" ? "border-primary text-primary" : "border-transparent text-on-surface-variant hover:text-on-surface"
          }`}
        >
          <span className="material-symbols-outlined text-[16px]">notifications</span>
          Notifications
          {unread > 0 && <span className="min-w-[16px] h-4 px-1 rounded-full bg-error text-on-primary text-[10px] font-semibold flex items-center justify-center">{unread}</span>}
        </button>
        <button
          onClick={() => setTab("alerts")}
          className={`flex items-center gap-sm px-md py-2.5 font-label-caps text-label-caps uppercase border-b-2 -mb-px transition-colors ${
            tab === "alerts" ? "border-primary text-primary" : "border-transparent text-on-surface-variant hover:text-on-surface"
          }`}
        >
          <span className="material-symbols-outlined text-[16px]">rule</span>
          Alerts
        </button>
      </div>
        </div>
      </div>

      {tab === "alerts" && (
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-lg">
        <div className="lg:col-span-8 flex flex-col gap-lg">
          <div className="glass-panel rounded-lg p-lg">
            <div className="flex justify-between items-center mb-md pb-md border-b border-outline-variant">
              <h2 className="font-label-caps text-label-caps text-on-surface flex items-center gap-2">
                <span className="material-symbols-outlined text-primary text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>rule</span>
                Active Rules
              </h2>
              <div className="relative">
                <span className="material-symbols-outlined absolute left-2 top-1.5 text-outline text-[16px]" style={{ fontVariationSettings: "'FILL' 0" }}>search</span>
                <input
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                  placeholder="Filter rules..."
                  type="text"
                  className="bg-surface-container-low border border-outline-variant text-body-sm rounded-DEFAULT py-1 pl-8 pr-3 focus:outline-none focus:border-primary focus:ring-0 w-48 text-on-surface"
                />
              </div>
            </div>
            <div className="flex flex-col gap-2">
              {filteredRules.length === 0 && (
                <p className="font-body-sm text-body-sm text-on-surface-variant p-md text-center">
                  {filter ? `No rules match "${filter}".` : "No alert rules yet. Click 'New Rule' to create one."}
                </p>
              )}
              {filteredRules.map((rule) => {
                const badge = severityBadge(rule.severity);
                const meta = METRIC_META[rule.metric] ?? { label: rule.metric, icon: "rule" };
                const target = rule.target_app ? apps?.find((a) => a.id === rule.target_app)?.name : undefined;
                return (
                  <div
                    key={rule.id}
                    className={`bg-surface-container-low border border-outline-variant rounded-DEFAULT p-md flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 hover:border-primary/50 transition-colors ${rule.enabled ? "" : "opacity-60"}`}
                  >
                    <div className="flex gap-4 items-start">
                      <div className={`w-8 h-8 rounded-full ${severityIconBg(rule.severity)} flex items-center justify-center shrink-0 mt-1`}>
                        <span className="material-symbols-outlined text-on-error-container text-[16px]" style={{ fontVariationSettings: "'FILL' 1" }}>{meta.icon}</span>
                      </div>
                      <div>
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="font-body-md text-body-md font-medium text-on-surface">{rule.name}</h3>
                          <span className={`font-code-md text-[10px] px-1.5 py-0.5 rounded-sm border ${badge.cls}`}>{badge.label}</span>
                        </div>
                        <p className="font-body-sm text-body-sm text-on-surface-variant mb-2">
                          Triggers when {meta.label.toLowerCase()} &gt; {rule.threshold}
                          {rule.metric === "cpu" ? "%" : rule.metric === "memory_pct" ? "%" : " MiB"} for {rule.window_s}s{target ? ` on ${target}` : ""}.
                        </p>
                        <div className="flex items-center gap-3">
                          <div className="flex items-center gap-1 text-on-surface-variant font-code-md text-[11px]">
                            <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 0" }}>tag</span>
                            {rule.metric}
                          </div>
                          <div className="flex items-center gap-1 text-on-surface-variant font-code-md text-[11px]">
                            <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 0" }}>notifications</span>
                            in-app
                          </div>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="flex items-center">
                        <input
                          checked={rule.enabled}
                          onChange={() => toggleRule(rule)}
                          className="hidden toggle-checkbox"
                          id={`tgl-${rule.id}`}
                          type="checkbox"
                        />
                        <label className="toggle-label" htmlFor={`tgl-${rule.id}`} />
                      </div>
                      <button className="text-outline-variant hover:text-on-surface transition-colors">
                        <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 0" }}>more_vert</span>
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        <div className="lg:col-span-4 flex flex-col gap-lg">
          <div className="glass-panel rounded-lg p-lg">
            <div className="flex justify-between items-center mb-md pb-md border-b border-outline-variant">
              <h2 className="font-label-caps text-label-caps text-on-surface flex items-center gap-2">
                <span className="material-symbols-outlined text-primary text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>hub</span>
                Channels
              </h2>
              <button className="text-primary hover:text-primary-fixed transition-colors">
                <span className="material-symbols-outlined text-[18px]" style={{ fontVariationSettings: "'FILL' 0" }}>add_circle</span>
              </button>
            </div>
            <div className="grid grid-cols-2 gap-sm">
              {(["slack", "discord", "email", "webhook"] as const).map((type) => {
                const label = type === "webhook" ? "Webhooks" : type.charAt(0).toUpperCase() + type.slice(1);
                const st = channelStatus(type);
                return (
                  <div
                    key={type}
                    className="bg-surface-container-low border border-outline-variant rounded-DEFAULT p-sm flex items-center gap-3 hover:border-primary/50 cursor-pointer transition-colors relative overflow-hidden group"
                  >
                    <div className="absolute inset-0 bg-gradient-to-br from-primary/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                    <ChannelIcon type={type} />
                    <div className="z-10">
                      <div className="font-body-sm text-body-sm font-medium">{label}</div>
                      <div className={`font-label-caps text-[9px] flex items-center gap-1 mt-0.5 ${st.connected ? "text-primary" : "text-outline"}`}>
                        {st.connected && <div className="w-1.5 h-1.5 bg-primary rounded-full" />}
                        {st.text}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="glass-panel rounded-lg p-0 flex-grow flex flex-col overflow-hidden">
            <div className="p-lg pb-sm flex justify-between items-center border-b border-outline-variant">
              <h2 className="font-label-caps text-label-caps text-on-surface flex items-center gap-2">
                <span className="material-symbols-outlined text-primary text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>history</span>
                Recent Activity
              </h2>
            </div>
            <div className="overflow-x-auto overflow-y-auto max-h-[300px]">
              {(events ?? []).length === 0 && (
                <p className="font-body-sm text-body-sm text-on-surface-variant p-lg text-center">No alert activity yet.</p>
              )}
              <table className="w-full text-left border-collapse">
                <tbody className="font-code-md text-code-md text-on-surface-variant divide-y divide-outline-variant/50">
                  {(events ?? []).map((e) => (
                    <tr key={e.id} className="table-row-hover transition-colors">
                      <td className="py-2 pl-lg pr-2 w-8">
                        <div className={`w-2 h-2 rounded-full ${e.severity === "critical" ? "bg-error" : e.severity === "warning" ? "bg-tertiary" : "bg-outline"}`} />
                      </td>
                      <td className="py-2 px-2 text-on-surface truncate max-w-[150px]">{e.message}</td>
                      <td className="py-2 pr-lg pl-2 text-right text-[11px] opacity-70">{timeAgo(e.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="p-sm text-center border-t border-outline-variant bg-surface-container-low/50">
              <button
                onClick={() => navigate({ to: "/monitoring" } as never)}
                className="font-label-caps text-label-caps text-primary hover:text-primary-fixed transition-colors"
              >
                View All Logs
              </button>
            </div>
          </div>
        </div>
      </div>

      )}

      {tab === "notifications" && (
        <div className="flex flex-col gap-lg">
          <div className="flex items-center justify-between">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Notification History</h2>
            <div className="flex items-center gap-sm">
              <label className="flex items-center gap-xs cursor-pointer select-none">
                <input type="checkbox" checked={showOnlyUnread} onChange={(e) => setShowOnlyUnread(e.target.checked)} className="w-3.5 h-3.5 rounded-sm bg-surface border-outline-variant text-primary" />
                <span className="font-body-sm text-body-sm text-on-surface-variant">Unread only</span>
              </label>
              {unread > 0 && (
                <Button variant="ghost" onClick={markAllRead}>Mark all read</Button>
              )}
            </div>
          </div>
          <div className="bg-surface-container-low border border-outline-variant rounded-lg divide-y divide-outline-variant/40 max-h-[70vh] overflow-y-auto sidebar-scroll">
            {notifList.length === 0 && (
              <p className="font-body-sm text-body-sm text-on-surface-variant p-sm">No notifications yet. Deploys, backups and servers will appear here in real time.</p>
            )}
            {(showOnlyUnread ? notifList.filter((n) => !n.read) : notifList).map((n) => {
              const meta = notifMeta(n.type);
              let parsed: Record<string, string> = {};
              try { parsed = JSON.parse(n.payload || "{}"); } catch {}
              return (
                <button
                  key={n.id}
                  onClick={() => {
                    if (!n.read) markRead(n.id);
                    const target = parsed.app_id || parsed.service_id;
                    if (target) navigate({ to: "/apps/$appId", params: { appId: target } } as never);
                  }}
                  className={`w-full flex items-start gap-sm px-md py-3 text-left hover:bg-surface-container-high/60 transition-colors ${n.read ? "" : "bg-primary/5"}`}
                >
                  <span className={`material-symbols-outlined text-[16px] shrink-0 mt-0.5 ${meta.color}`}>{meta.icon}</span>
                  <span className="flex-1 min-w-0">
                    <span className={`font-body-sm text-body-sm line-clamp-5 break-words ${n.read ? "text-on-surface-variant" : "text-on-surface font-medium"}`} title={n.message}>{n.message}</span>
                    <span className="block font-code-md text-code-md text-on-surface-variant/50 mt-0.5">{n.type} · {timeAgo(n.created_at)}</span>
                  </span>
                  {!n.read && <span className="w-1.5 h-1.5 rounded-full bg-primary shrink-0 mt-2" />}
                </button>
              );
            })}
          </div>
        </div>
      )}

      <Modal open={ruleOpen} onClose={() => setRuleOpen(false)} title="New alert rule">
        <div className="space-y-lg">
          <Field label="Rule name">
            <Input icon="label" placeholder="e.g. High CPU Utilization" value={name} onChange={(e) => setName(e.target.value)} />
          </Field>
          <div className="grid grid-cols-2 gap-lg">
            <Field label="Metric">
              <Select value={metric} onChange={(e) => setMetric(e.target.value)}>
                <option value="cpu">CPU (%)</option>
                <option value="memory">Memory (MiB)</option>
                <option value="memory_pct">Memory (%)</option>
              </Select>
            </Field>
            <Field label="Threshold">
              <Input icon="speed" type="number" placeholder="80" value={threshold} onChange={(e) => setThreshold(e.target.value)} />
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-lg">
            <Field label="Severity">
              <Select value={severity} onChange={(e) => setSeverity(e.target.value)}>
                <option value="warning">warning</option>
                <option value="critical">critical</option>
                <option value="info">info</option>
              </Select>
            </Field>
            <Field label="Target service">
              <Select value={targetApp} onChange={(e) => setTargetApp(e.target.value)}>
                <option value="">All services</option>
                {(apps ?? []).map((a) => (
                  <option key={a.id} value={a.id}>{a.name}</option>
                ))}
              </Select>
            </Field>
          </div>
          <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
            <Button variant="ghost" onClick={() => setRuleOpen(false)}>Cancel</Button>
            <Button onClick={addRule}>Create rule</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

export default Notifications;

export const Route = createFileRoute("/_shell/notifications/")({
  component: Notifications,
});
