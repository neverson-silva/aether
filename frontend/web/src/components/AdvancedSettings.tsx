import { Accordion, Checkbox, Input, Slider } from "@aether/design-system";

function fmtGB(mb: number): string {
  if (mb === 0) return "∞";
  if (mb % 1024 === 0) return `${mb / 1024} GB`;
  return `${(mb / 1024).toFixed(1)} GB`;
}

function sliderValue(value: number | number[]): number {
  return Array.isArray(value) ? value[0] ?? 0 : value;
}

function fmtStorage(mb: number): string {
  if (mb === 0) return "Unlimited storage";
  return `${mb / 1024} GB storage`;
}

export interface AdvancedSettingsValues {
  cpu: string;
  memMB: number;
  storageMB: number;
  healthEnabled: boolean;
  healthPath: string;
}

export function AdvancedSettings({
  values,
  onChange,
  showHealth = true,
}: {
  values: AdvancedSettingsValues;
  onChange: (v: AdvancedSettingsValues) => void;
  showHealth?: boolean;
}) {
  const patch = (p: Partial<AdvancedSettingsValues>) => onChange({ ...values, ...p });

  return (
    <Accordion
      items={[{
        value: "advanced-settings",
        title: (
          <span className="flex w-full items-center gap-sm">
            Advanced settings
            <span className="ml-auto font-code-md text-code-md font-normal text-muted-foreground">
              {values.memMB > 0 ? fmtGB(values.memMB).replace(" GB", "GB RAM") : "∞ RAM"} · {values.cpu} CPU · {fmtGB(values.storageMB)} storage
            </span>
          </span>
        ),
        content: (
          <div className="space-y-lg">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
          <div>
            <Slider
              label="CPU limit"
              description={`${values.cpu} CPU allocated`}
              min={0.25}
              max={8}
              step={0.25}
              value={Number.parseFloat(values.cpu) || 0.5}
              onValueChange={(value) => patch({ cpu: sliderValue(value).toString() })}
            />
          </div>
          <div>
            <Slider
              label="Memory (RAM)"
              description={fmtGB(values.memMB)}
              min={256}
              max={8192}
              step={256}
              value={Math.min(8192, Math.max(256, values.memMB))}
              onValueChange={(value) => patch({ memMB: sliderValue(value) })}
            />
          </div>
        </div>

        <div>
          <Slider
            label="Storage limit (max volume size)"
            description={fmtStorage(values.storageMB)}
            min={0}
            max={102400}
            step={1024}
            value={Math.min(102400, Math.max(0, values.storageMB))}
            onValueChange={(value) => {
              patch({ storageMB: sliderValue(value) });
            }}
          />
          <p className="font-code-md text-code-md text-on-surface-variant/60 mt-sm">
            Maximum quota applied to the persistent volume. The minimum value means unlimited.
          </p>
        </div>

        {showHealth && (
          <div className="flex items-center gap-sm">
            <Checkbox checked={values.healthEnabled} onCheckedChange={(checked) => patch({ healthEnabled: checked === true })} />
            <span className="font-body-md text-body-md text-on-surface">Enable health check</span>
            {values.healthEnabled && (
              <Input className="max-w-[200px]" placeholder="/health" value={values.healthPath} onChange={(e) => patch({ healthPath: e.target.value })} />
            )}
          </div>
        )}
          </div>
        ),
      }]}
    />
  );
}
