import { useEffect, useMemo, useState } from "react";
import type { BackupConfig, BackupSchedule, S3Destination } from "../../../../api/types";
import { useUpsertDatabaseBackupConfig } from "../../../../hooks";
import { Button, Dialog, Field, Input, NativeSelect, TimePicker } from "@aether/design-system";

const TIMEZONES: string[] = (() => {
  try {
    return (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf?.("timeZone") ?? ["UTC", "America/Sao_Paulo", "America/New_York", "Europe/London"];
  } catch {
    return ["UTC", "America/Sao_Paulo", "America/New_York", "Europe/London"];
  }
})();

const SCHEDULE_TYPES: { id: BackupSchedule["type"]; label: string }[] = [
  { id: "hourly", label: "Every hour" },
  { id: "daily", label: "Every day" },
  { id: "weekly", label: "Every week" },
  { id: "biweekly", label: "Every 15 days" },
  { id: "custom", label: "Custom" },
];

const WEEKDAYS = ["sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"];

function cronValid(cron: string): boolean {
  return /^(\*|[0-5]?\d)(\s+(\*|[0-5]?\d)){4}$/.test(cron.trim());
}

function describeSchedule(s: BackupSchedule): string {
  switch (s.type) {
    case "hourly":
      return `Every hour at :${String(s.minute ?? 0).padStart(2, "0")}`;
    case "daily":
      return `Every day at ${s.at ?? "03:00"}`;
    case "weekly":
      return `Every ${s.day_of_week} at ${s.at ?? "03:00"}`;
    case "biweekly":
      return `Every 15 days from ${s.start_date} at ${s.at ?? "03:00"}`;
    case "custom":
      return `Cron ${s.cron} (${s.timezone})`;
    default:
      return "Every day";
  }
}

export function describeScheduleExport(s: BackupSchedule): string {
  return describeSchedule(s);
}

export function BackupConfigDialog({
  dbId,
  existing,
  destinations,
  onClose,
}: {
  dbId: string;
  existing: BackupConfig | null;
  destinations: S3Destination[];
  onClose: () => void;
}) {
  const save = useUpsertDatabaseBackupConfig(dbId);
  const [destinationId, setDestinationId] = useState(existing?.destination_id ?? destinations[0]?.id ?? "");
  const [pathPrefix, setPathPrefix] = useState(existing?.path_prefix ?? "databases");
  const [scheduleType, setScheduleType] = useState<BackupSchedule["type"]>(existing?.schedule.type ?? "daily");
  const [minute, setMinute] = useState(existing?.schedule.minute ?? 0);
  const [at, setAt] = useState(existing?.schedule.at ?? "03:00");
  const [day, setDay] = useState(existing?.schedule.day_of_week ?? "sunday");
  const [startDate, setStartDate] = useState(existing?.schedule.start_date ?? new Date().toISOString().slice(0, 10));
  const [cron, setCron] = useState(existing?.schedule.cron ?? "0 3 * * *");
  const [timezone, setTimezone] = useState(existing?.schedule.timezone ?? "UTC");
  const [retention, setRetention] = useState<"all" | "latest">(existing?.retention.type ?? "all");
  const [saving, setSaving] = useState(false);

  const cronOk = scheduleType !== "custom" || cronValid(cron);

  const selectedDest = useMemo(() => destinations.find((d) => d.id === destinationId), [destinationId, destinations]);

  const submit = async () => {
    setSaving(true);
    try {
      await save.mutateAsync({
        id: existing?.id,
        enabled: true,
        destination_id: destinationId,
        path_prefix: pathPrefix,
        schedule: { type: scheduleType, minute, at, day_of_week: day, start_date: startDate, cron, timezone },
        retention: { type: retention },
      } as Partial<BackupConfig>);
      onClose();
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open trigger={<span />} onOpenChange={(open) => { if (!open) onClose(); }} title={existing ? "Edit backup configuration" : "Configure backup"}>
      <div className="space-y-lg">
        <div className="space-y-md">
          <Field label="Backup destination">
            <NativeSelect value={destinationId} onChange={(e) => setDestinationId(e.target.value)} options={destinations.length === 0 ? [{ value: "", label: "No S3 destinations available" }] : destinations.map((d) => ({ value: d.id, label: d.name }))} />
          </Field>
          {selectedDest && (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-md px-md py-sm rounded border border-outline-variant/60">
              <div>
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Bucket</span>
                <span className="font-code-md text-code-md text-on-surface">{selectedDest.bucket}</span>
              </div>
              <div>
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Region</span>
                <span className="font-code-md text-code-md text-on-surface">{selectedDest.region || "—"}</span>
              </div>
              <div className="truncate">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase block mb-0.5">Endpoint</span>
                <span className="font-code-md text-code-md text-on-surface truncate">{selectedDest.endpoint}</span>
              </div>
            </div>
          )}
          <Field label="Path prefix">
            <Input value={pathPrefix} onChange={(e) => setPathPrefix(e.target.value)} placeholder="databases/production" />
          </Field>
        </div>

        <div className="space-y-md">
          <Field label="Backup schedule">
            <div className="flex flex-wrap gap-sm">
              {SCHEDULE_TYPES.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => setScheduleType(t.id)}
                  className={`px-md py-1.5 rounded font-label-caps text-label-caps uppercase border transition-colors ${
                    scheduleType === t.id ? "border-primary text-primary bg-primary/10" : "border-outline-variant text-on-surface-variant hover:text-on-surface"
                  }`}
                >
                  {t.label}
                </button>
              ))}
            </div>
          </Field>

          {scheduleType === "hourly" && (
            <Field label="Run at">
              <div className="flex items-center gap-sm">
                <Input type="number" min={0} max={59} value={minute} onChange={(e) => setMinute(Number(e.target.value))} className="w-24" />
                <span className="font-body-sm text-body-sm text-on-surface-variant">minutes past every hour</span>
              </div>
            </Field>
          )}

          {(scheduleType === "daily" || scheduleType === "weekly" || scheduleType === "biweekly") && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-md">
              {scheduleType === "weekly" && (
                <Field label="Day">
                  <NativeSelect value={day} onChange={(e) => setDay(e.target.value)} options={WEEKDAYS.map((d) => ({ value: d, label: d[0].toUpperCase() + d.slice(1) }))} />
                </Field>
              )}
              {scheduleType === "biweekly" && (
                <Field label="Starting from">
                  <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
                </Field>
              )}
              <Field label="Time">
                <TimePicker value={at} onChange={(event) => setAt(event.target.value)} />
              </Field>
            </div>
          )}

          {scheduleType === "custom" && (
            <Field label="Cron expression" description={cronOk ? "Valid 5-field crontab expression." : "Invalid cron expression. Expected a valid 5-field crontab expression."}>
              <Input value={cron} onChange={(e) => setCron(e.target.value)} placeholder="0 3 * * *" className={cronOk ? "" : "border-error"} />
            </Field>
          )}

          <Field label="Timezone">
            <NativeSelect value={timezone} onChange={(e) => setTimezone(e.target.value)} options={TIMEZONES.map((tz) => ({ value: tz, label: tz }))} />
          </Field>
        </div>

        <Field label="Retention">
          <div className="flex flex-col gap-sm">
            <label className="flex items-center gap-sm p-sm rounded border border-outline-variant/60 cursor-pointer">
              <input type="radio" checked={retention === "all"} onChange={() => setRetention("all")} />
              <span className="font-body-md text-body-md text-on-surface">Keep all backups</span>
            </label>
            <label className="flex items-center gap-sm p-sm rounded border border-outline-variant/60 cursor-pointer">
              <input type="radio" checked={retention === "latest"} onChange={() => setRetention("latest")} />
              <span className="font-body-md text-body-md text-on-surface">Keep only the latest backup</span>
            </label>
          </div>
        </Field>

        <div className="flex items-center justify-end gap-sm border-t border-outline-variant pt-md">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={() => void submit()} loading={saving} disabled={!destinationId || !cronOk}>
            Save configuration
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
