import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import {
  Bell,
  BellRinging,
  CheckCircle,
  Clock,
  DotsThreeVertical,
  EnvelopeSimple,
  Funnel,
  Gear,
  HardDrives,
  ListChecks,
  MagnifyingGlass,
  Plus,
  Pulse,
  Tag,
  WebhooksLogo,
  XCircle,
} from "@phosphor-icons/react";
import type { Icon as DesignIcon } from "@aether/design-system";
import {
  Badge,
  Button,
  Card,
  Checkbox,
  Dialog,
  EmptyState,
  Field,
  IconButton,
  InlineError,
  Input,
  NativeSelect,
  Skeleton,
  useToast,
} from "@aether/design-system";
import {
  useAlertEvents,
  useAlertRules,
  useApps,
  useCreateAlertRule,
  useNotificationChannels,
  useOutWebhooks,
  useSetAlertRuleEnabled,
} from "../../../hooks";
import type { AlertRule } from "../../../hooks";
import { useNotifications as useProviderNotifications } from "../../../components/NotificationProvider";

const designIcon = (icon: typeof Pulse) => icon as unknown as DesignIcon;
const METRIC_META: Record<string, { label: string; icon: typeof Pulse }> = {
  cpu: { label: "CPU", icon: Pulse },
  memory: { label: "Memory", icon: HardDrives },
  memory_pct: { label: "Memory %", icon: HardDrives },
  disk: { label: "Disk", icon: HardDrives },
};

function timeAgo(iso: string) {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

function notificationSummary(message: string) {
  const compact = message.replace(/\s+/g, " ").trim();
  return compact.length > 240 ? `${compact.slice(0, 240)}…` : compact;
}
function notificationTone(type: string) {
  if (type.includes("failed"))
    return { icon: XCircle, tone: "danger" as const };
  if (type.includes("ready") || type.includes("finished"))
    return { icon: CheckCircle, tone: "success" as const };
  if (
    type.includes("queued") ||
    type.includes("building") ||
    type.includes("starting") ||
    type.includes("healthcheck")
  )
    return { icon: Clock, tone: "warning" as const };
  if (type.includes("server"))
    return { icon: HardDrives, tone: "info" as const };
  if (type.includes("alert"))
    return { icon: BellRinging, tone: "danger" as const };
  return { icon: Bell, tone: "neutral" as const };
}
function severityTone(severity: string) {
  if (severity === "critical") return "danger" as const;
  if (severity === "warning") return "warning" as const;
  return "info" as const;
}
function LoadingState() {
  return (
    <div
      className="flex min-h-32 items-center justify-center"
      role="status"
      aria-label="Loading"
    >
      <Skeleton variant="card" className="max-w-xl" />
    </div>
  );
}

function Notifications() {
  const rulesQuery = useAlertRules();
  const channelsQuery = useNotificationChannels();
  const webhooksQuery = useOutWebhooks();
  const eventsQuery = useAlertEvents(20);
  const appsQuery = useApps();
  const createRule = useCreateAlertRule();
  const setEnabled = useSetAlertRuleEnabled();
  const { add } = useToast();
  const navigate = useNavigate();
  const { list, unread, markRead, markAllRead } = useProviderNotifications();
  const [tab, setTab] = useState<"notifications" | "alerts">("notifications");
  const [onlyUnread, setOnlyUnread] = useState(false);
  const [filter, setFilter] = useState("");
  const [ruleOpen, setRuleOpen] = useState(false);
  const [name, setName] = useState("");
  const [metric, setMetric] = useState("cpu");
  const [threshold, setThreshold] = useState("80");
  const [severity, setSeverity] = useState("warning");
  const [targetApp, setTargetApp] = useState("");
  const rules = rulesQuery.data ?? [];
  const filteredRules = rules.filter(
    (rule) =>
      !filter ||
      `${rule.name} ${rule.metric}`
        .toLowerCase()
        .includes(filter.toLowerCase()),
  );
  const visibleNotifications = onlyUnread
    ? list.filter((item) => !item.read)
    : list;
  const loading =
    rulesQuery.isLoading ||
    channelsQuery.isLoading ||
    webhooksQuery.isLoading ||
    eventsQuery.isLoading ||
    appsQuery.isLoading;
  const queryError =
    rulesQuery.error ||
    channelsQuery.error ||
    webhooksQuery.error ||
    eventsQuery.error ||
    appsQuery.error;
  const addRule = async () => {
    const parsedThreshold = Number.parseFloat(threshold);
    if (!name.trim() || !Number.isFinite(parsedThreshold)) {
      add({
        title: "Invalid rule",
        description: "Enter a name and a valid threshold.",
        tone: "error",
      });
      return;
    }
    try {
      await createRule.mutateAsync({
        name: name.trim(),
        metric,
        threshold: parsedThreshold,
        severity,
        target_app: targetApp,
        window_s: 30,
        enabled: true,
      });
      setRuleOpen(false);
      setName("");
      setThreshold("80");
      add({ title: "Alert rule created", tone: "success" });
    } catch (error) {
      add({
        title: "Could not create alert rule",
        description:
          error instanceof Error ? error.message : "Try again later.",
        tone: "error",
      });
    }
  };
  const toggleRule = (rule: AlertRule) => {
    setEnabled.mutate({ id: rule.id, enabled: !rule.enabled });
    add({
      title: rule.enabled ? "Alert rule disabled" : "Alert rule enabled",
      description: rule.name,
      tone: rule.enabled ? "warning" : "success",
    });
  };
  return (
    <div className="mx-auto max-w-[1600px] space-y-6">
      <header className="flex flex-col items-start justify-between gap-4 md:flex-row md:items-end">
        <div>
          <h1 className="text-headline-sm font-semibold text-foreground">
            Notifications &amp; Alerts
          </h1>
          <p className="mt-1 text-body-md text-muted-foreground">
            Notification history, alert rules, routing and communication
            channels.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="secondary"
            icon={designIcon(Clock)}
            onClick={() => navigate({ to: "/monitoring" } as never)}
          >
            View audit log
          </Button>
          <Dialog
            open={ruleOpen}
            onOpenChange={setRuleOpen}
            title="Create alert rule"
            description="Define when Aether should notify your team."
            trigger={<Button icon={designIcon(Plus)}>New rule</Button>}
          >
            <div className="space-y-5">
              <Field label="Rule name" required>
                <Input
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder="High CPU utilization"
                  leadingIcon={designIcon(Tag)}
                />
              </Field>
              <div className="grid gap-4 sm:grid-cols-2">
                <NativeSelect
                  aria-label="Metric"
                  value={metric}
                  onChange={(event) => setMetric(event.target.value)}
                  options={[
                    { label: "CPU (%)", value: "cpu" },
                    { label: "Memory (MiB)", value: "memory" },
                    { label: "Memory (%)", value: "memory_pct" },
                  ]}
                />
                <Field label="Threshold" required>
                  <Input
                    type="number"
                    value={threshold}
                    onChange={(event) => setThreshold(event.target.value)}
                    leadingIcon={designIcon(Funnel)}
                  />
                </Field>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <NativeSelect
                  aria-label="Severity"
                  value={severity}
                  onChange={(event) => setSeverity(event.target.value)}
                  options={[
                    { label: "Warning", value: "warning" },
                    { label: "Critical", value: "critical" },
                    { label: "Info", value: "info" },
                  ]}
                />
                <NativeSelect
                  aria-label="Target service"
                  value={targetApp}
                  onChange={(event) => setTargetApp(event.target.value)}
                  options={[
                    { label: "All services", value: "" },
                    ...(appsQuery.data ?? []).map((app) => ({
                      label: app.name,
                      value: app.id,
                    })),
                  ]}
                />
              </div>
              <div className="flex justify-end gap-2 border-t border-border pt-4">
                <Button variant="ghost" onClick={() => setRuleOpen(false)}>
                  Cancel
                </Button>
                <Button loading={createRule.isPending} onClick={addRule}>
                  Create rule
                </Button>
              </div>
            </div>
          </Dialog>
        </div>
      </header>
      {queryError ? (
        <InlineError
          title="Could not load notification data"
          message="Some sections may be incomplete. Try again to refresh the page."
          onRetry={() => window.location.reload()}
        />
      ) : null}
      <div
        className="flex gap-1 border-b border-border"
        role="tablist"
        aria-label="Notification views"
      >
        <button
          type="button"
          role="tab"
          aria-selected={tab === "notifications"}
          onClick={() => setTab("notifications")}
          className={`flex items-center gap-2 border-b-2 px-4 py-3 text-body-sm font-semibold ${tab === "notifications" ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"}`}
        >
          <Bell size={17} />
          Notifications{unread ? <Badge tone="danger">{unread}</Badge> : null}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === "alerts"}
          onClick={() => setTab("alerts")}
          className={`flex items-center gap-2 border-b-2 px-4 py-3 text-body-sm font-semibold ${tab === "alerts" ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"}`}
        >
          <ListChecks size={17} />
          Alert rules
        </button>
      </div>
      {tab === "notifications" ? (
        <section
          className="space-y-4"
          aria-labelledby="notification-history-heading"
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2
              id="notification-history-heading"
              className="text-body-sm font-semibold uppercase tracking-wide text-muted-foreground"
            >
              Notification history
            </h2>
            <div className="flex items-center gap-3">
              <Checkbox
                label="Unread only"
                checked={onlyUnread}
                onCheckedChange={(checked) => setOnlyUnread(checked === true)}
              />
              {unread ? (
                <Button variant="ghost" size="sm" onClick={markAllRead}>
                  Mark all read
                </Button>
              ) : null}
            </div>
          </div>
          {loading ? (
            <LoadingState />
          ) : visibleNotifications.length ? (
            <div className="space-y-2">
              {visibleNotifications.map((notification) => {
                const meta = notificationTone(notification.type);
                const Icon = meta.icon;
                let payload: Record<string, string> = {};
                try {
                  payload = JSON.parse(notification.payload || "{}");
                } catch {
                  payload = {};
                }
                const target = payload.service_id || payload.app_id;
                return (
                  <Card
                    key={notification.id}
                    variant="interactive"
                    padding="md"
                    className={
                      !notification.read ? "border-primary/40 bg-primary/5" : ""
                    }
                  >
                    <div className="flex items-start gap-3">
                      <Icon
                        size={20}
                        className="mt-0.5 shrink-0 text-primary"
                        aria-hidden="true"
                      />
                      <button
                        type="button"
                        title={notification.message}
                        className="min-w-0 flex-1 text-left"
                        onClick={() => {
                          if (!notification.read) markRead(notification.id);
                          if (target)
                            navigate({
                              to: "/apps/$appId",
                              params: { appId: target },
                            } as never);
                        }}
                      >
                        <span
                          className={`line-clamp-3 text-body-md ${notification.read ? "text-muted-foreground" : "font-semibold text-foreground"}`}
                        >
                          {notificationSummary(notification.message)}
                        </span>
                        <span className="mt-1 block font-mono text-code-md text-muted-foreground">
                          {notification.type} ·{" "}
                          {timeAgo(notification.created_at)}
                        </span>
                      </button>
                      {!notification.read ? (
                        <span
                          className="mt-2 size-2 shrink-0 rounded-full bg-primary"
                          aria-label="Unread"
                        />
                      ) : null}
                    </div>
                  </Card>
                );
              })}
            </div>
          ) : (
            <EmptyState
              icon={designIcon(Bell)}
              title={
                onlyUnread ? "No unread notifications" : "No notifications yet"
              }
              description="Deploys, backups and services will appear here in real time."
            />
          )}
        </section>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1.6fr)_minmax(20rem,1fr)]">
          <Card
            header={
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <ListChecks size={20} className="text-primary" />
                  <h2 className="text-body-md font-semibold">Active rules</h2>
                </div>
                <Input
                  aria-label="Filter rules"
                  value={filter}
                  onChange={(event) => setFilter(event.target.value)}
                  placeholder="Filter rules"
                  leadingIcon={designIcon(MagnifyingGlass)}
                  size="sm"
                />
              </div>
            }
          >
            {rulesQuery.isLoading ? (
              <LoadingState />
            ) : filteredRules.length ? (
              <div className="space-y-2">
                {filteredRules.map((rule) => {
                  const meta = METRIC_META[rule.metric] ?? {
                    label: rule.metric,
                    icon: ListChecks,
                  };
                  const Icon = meta.icon;
                  return (
                    <div
                      key={rule.id}
                      className={`flex flex-col justify-between gap-4 rounded-lg border border-border bg-surface-card p-4 sm:flex-row sm:items-center ${rule.enabled ? "" : "opacity-60"}`}
                    >
                      <div className="flex min-w-0 items-start gap-3">
                        <Icon
                          size={20}
                          className="mt-1 shrink-0 text-primary"
                        />
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <h3 className="truncate font-semibold text-foreground">
                              {rule.name}
                            </h3>
                            <Badge tone={severityTone(rule.severity)}>
                              {rule.severity}
                            </Badge>
                          </div>
                          <p className="mt-1 text-body-sm text-muted-foreground">
                            Triggers when {meta.label.toLowerCase()} &gt;{" "}
                            {rule.threshold}
                            {rule.metric === "cpu" ||
                            rule.metric === "memory_pct"
                              ? "%"
                              : " MiB"}{" "}
                            for {rule.window_s}s
                            {rule.target_app
                              ? ` on ${appsQuery.data?.find((app) => app.id === rule.target_app)?.name ?? "a service"}`
                              : ""}
                            .
                          </p>
                          <div className="mt-2 flex gap-3 font-mono text-code-md text-muted-foreground">
                            <span>
                              <Tag size={14} className="mr-1 inline" />
                              {rule.metric}
                            </span>
                            <span>
                              <Bell size={14} className="mr-1 inline" />
                              in-app
                            </span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Checkbox
                          aria-label={`${rule.enabled ? "Disable" : "Enable"} ${rule.name}`}
                          checked={rule.enabled}
                          onCheckedChange={() => toggleRule(rule)}
                        />
                        <IconButton
                          icon={designIcon(DotsThreeVertical)}
                          label={`Actions for ${rule.name}`}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <EmptyState
                icon={designIcon(ListChecks)}
                title={filter ? "No matching rules" : "No alert rules yet"}
                description={
                  filter
                    ? "Try a different filter."
                    : "Create a rule to start monitoring your services."
                }
                action={
                  <Button
                    icon={designIcon(Plus)}
                    onClick={() => setRuleOpen(true)}
                  >
                    New rule
                  </Button>
                }
              />
            )}
          </Card>
          <div className="space-y-6">
            <Card
              header={
                <div className="flex items-center gap-2">
                  <Gear size={20} className="text-primary" />
                  <h2 className="text-body-md font-semibold">
                    Notification channels
                  </h2>
                </div>
              }
            >
              {channelsQuery.isLoading || webhooksQuery.isLoading ? (
                <LoadingState />
              ) : (
                <div className="grid grid-cols-2 gap-3">
                  {(["slack", "discord", "email", "webhook"] as const).map(
                    (type) => {
                      const count =
                        channelsQuery.data?.filter(
                          (channel) => channel.type === type && channel.enabled,
                        ).length ?? 0;
                      const webhookCount =
                        type === "webhook"
                          ? (webhooksQuery.data?.length ?? 0)
                          : 0;
                      const Icon =
                        type === "email"
                          ? EnvelopeSimple
                          : type === "webhook"
                            ? WebhooksLogo
                            : Bell;
                      const status = count
                        ? "Connected"
                        : webhookCount
                          ? `${webhookCount} active`
                          : channelsQuery.data?.some(
                                (channel) => channel.type === type,
                              )
                            ? "Configured"
                            : "Not set up";
                      return (
                        <div
                          key={type}
                          className="rounded-lg border border-border bg-surface-card p-3"
                        >
                          <Icon size={20} className="text-primary" />
                          <div className="mt-2 font-semibold capitalize text-foreground">
                            {type}
                          </div>
                          <div className="mt-1 text-label-caps text-muted-foreground">
                            {status}
                          </div>
                        </div>
                      );
                    },
                  )}
                </div>
              )}
            </Card>
            <Card
              header={
                <div className="flex items-center gap-2">
                  <Clock size={20} className="text-primary" />
                  <h2 className="text-body-md font-semibold">
                    Recent alert activity
                  </h2>
                </div>
              }
            >
              {eventsQuery.isLoading ? (
                <LoadingState />
              ) : eventsQuery.data?.length ? (
                <div className="space-y-3">
                  {eventsQuery.data.map((event) => (
                    <div key={event.id} className="flex gap-3">
                      <span
                        className={`mt-1.5 size-2 shrink-0 rounded-full ${event.severity === "critical" ? "bg-status-danger" : event.severity === "warning" ? "bg-status-warning" : "bg-status-info"}`}
                      />
                      <div className="min-w-0">
                        <p className="text-body-sm text-foreground">
                          {event.message}
                        </p>
                        <p className="mt-1 font-mono text-code-md text-muted-foreground">
                          {timeAgo(event.created_at)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState
                  icon={designIcon(Clock)}
                  title="No alert activity"
                  description="Triggered rules will appear here."
                />
              )}
            </Card>
          </div>
        </div>
      )}
    </div>
  );
}
export default Notifications;
export const Route = createFileRoute("/_shell/notifications/")({
  component: Notifications,
});
