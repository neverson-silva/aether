import { useState } from "react";
import { Input } from "./ui";

const CPU_PRESETS = ["0.25", "0.5", "1", "2", "4", "8"];
const MEM_PRESETS = [
  { label: "256 MB", value: 256 },
  { label: "512 MB", value: 512 },
  { label: "1 GB", value: 1024 },
  { label: "2 GB", value: 2048 },
  { label: "4 GB", value: 4096 },
  { label: "8 GB", value: 8192 },
  { label: "∞", value: 0 },
];
const STORAGE_PRESETS = [
  { label: "1 GB", value: 1024 },
  { label: "5 GB", value: 5120 },
  { label: "10 GB", value: 10240 },
  { label: "50 GB", value: 51200 },
  { label: "100 GB", value: 102400 },
  { label: "∞", value: 0 },
];

function fmtGB(mb: number): string {
  if (mb === 0) return "∞";
  if (mb % 1024 === 0) return `${mb / 1024} GB`;
  return `${(mb / 1024).toFixed(1)} GB`;
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
  const [storageCustom, setStorageCustom] = useState("");
  const patch = (p: Partial<AdvancedSettingsValues>) => onChange({ ...values, ...p });

  return (
    <details className="group border border-outline-variant rounded-lg">
      <summary className="flex items-center gap-sm px-md py-2.5 font-label-caps text-label-caps text-on-surface-variant uppercase cursor-pointer select-none hover:bg-surface-container-high/40 transition-colors">
        <span className="material-symbols-outlined text-[16px] transition-transform group-open:rotate-180">expand_more</span>
        Advanced Settings
        <span className="ml-auto font-code-md text-code-md text-on-surface-variant/60 normal-case">
          {values.memMB > 0 ? fmtGB(values.memMB).replace(" GB", "GB RAM") : "∞ RAM"} · {values.cpu} CPU · {fmtGB(values.storageMB)} storage
        </span>
      </summary>
      <div className="px-md pb-md pt-lg space-y-lg border-t border-outline-variant/60">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
          <div>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">CPU limit</p>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-sm">
              {CPU_PRESETS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => patch({ cpu: c })}
                  className={`px-sm py-2 rounded border font-code-md text-code-md transition-colors ${values.cpu === c ? "border-primary bg-primary/10 text-primary" : "border-outline-variant text-on-surface-variant hover:border-primary/40"}`}
                >
                  {c}
                </button>
              ))}
            </div>
            <Input icon="speed" className="mt-sm" placeholder="Custom (0.5, 500m, 2)" value={values.cpu} onChange={(e) => patch({ cpu: e.target.value })} />
          </div>
          <div>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Memory (RAM)</p>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-sm">
              {MEM_PRESETS.map((m) => (
                <button
                  key={m.label}
                  type="button"
                  onClick={() => patch({ memMB: m.value })}
                  className={`px-sm py-2 rounded border font-code-md text-code-md transition-colors ${values.memMB === m.value ? "border-primary bg-primary/10 text-primary" : "border-outline-variant text-on-surface-variant hover:border-primary/40"}`}
                >
                  {m.label}
                </button>
              ))}
            </div>
          </div>
        </div>

        <div>
          <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Storage limit (max volume size)</p>
          <div className="grid grid-cols-3 md:grid-cols-6 gap-sm">
            {STORAGE_PRESETS.map((s) => (
              <button
                key={s.label}
                type="button"
                onClick={() => {
                  patch({ storageMB: s.value });
                  setStorageCustom("");
                }}
                className={`px-sm py-2 rounded border font-code-md text-code-md transition-colors ${values.storageMB === s.value ? "border-primary bg-primary/10 text-primary" : "border-outline-variant text-on-surface-variant hover:border-primary/40"}`}
              >
                {s.label}
              </button>
            ))}
          </div>
          <Input
            icon="storage"
            className="mt-sm max-w-[220px]"
            placeholder="Custom (e.g. 20000 = 20GB)"
            value={storageCustom}
            onChange={(e) => {
              setStorageCustom(e.target.value);
              const n = parseInt(e.target.value, 10);
              if (isFinite(n) && n > 0) patch({ storageMB: n });
            }}
          />
          <p className="font-code-md text-code-md text-on-surface-variant/60 mt-sm">
            Maximum quota applied to the persistent volume (storage-opt size). 0 = unlimited.
          </p>
        </div>

        {showHealth && (
          <div className="flex items-center gap-sm">
            <input
              type="checkbox"
              checked={values.healthEnabled}
              onChange={(e) => patch({ healthEnabled: e.target.checked })}
              className="w-4 h-4 rounded-sm bg-surface border-outline-variant text-primary"
            />
            <span className="font-body-md text-body-md text-on-surface">Enable health check</span>
            {values.healthEnabled && (
              <Input icon="monitor_heart" className="max-w-[200px]" placeholder="/health" value={values.healthPath} onChange={(e) => patch({ healthPath: e.target.value })} />
            )}
          </div>
        )}
      </div>
    </details>
  );
}
