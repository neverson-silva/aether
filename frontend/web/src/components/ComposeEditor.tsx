import { useMemo, useState } from "react";
import { CheckCircle, Database, Warning, XCircle } from "@phosphor-icons/react";
import { Badge, Skeleton } from "@aether/design-system";
import { useValidateCompose } from "../hooks";

const KEYWORDS = ["services", "image", "build", "ports", "environment", "volumes", "networks", "depends_on", "restart", "command", "entrypoint", "container_name", "environment:", "version", "secrets", "configs", "deploy", "replicas", "healthcheck", "test", "interval", "timeout", "retries", "labels", "hostname", "working_dir"];

export function highlightYAML(line: string): React.ReactNode[] {
  const out: React.ReactNode[] = [];
  const parts = line.split(/(:[ \t]|^\s*[-])/g);
  for (const part of parts) {
    const trimmed = part.trim();
    const kw = KEYWORDS.find((k) => trimmed === k || trimmed.startsWith(k));
    if (kw && trimmed.length <= 30) {
      out.push(<span key={out.length} className="text-[#60a5fa]">{part}</span>);
    } else if (trimmed.startsWith("${")) {
      out.push(<span key={out.length} className="text-[#fbbf24]">{part}</span>);
    } else if (/^#/.test(trimmed)) {
      out.push(<span key={out.length} className="text-on-surface-variant/40 italic">{part}</span>);
    } else {
      out.push(<span key={out.length} className="text-on-surface">{part}</span>);
    }
  }
  return out;
}

export function ComposeEditor({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [line] = useState(0);
  const validation = useValidateCompose(value);
  void line;

  const lines = useMemo(() => value.split("\n"), [value]);

  return (
    <div className="grid grid-cols-1 xl:grid-cols-3 gap-md">
      <div className="xl:col-span-2">
        <div className="flex items-center justify-between mb-sm">
          <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">docker-compose.yml</p>
          <Badge tone={validation.data ? (validation.data.valid ? "success" : "danger") : "neutral"}>{validation.data ? (validation.data.valid ? "Valid" : "Invalid") : "Pending"}</Badge>
        </div>
        <div className="bg-surface-container-lowest border border-outline-variant rounded-lg overflow-hidden">
          <div className="flex items-center justify-between px-sm py-1.5 border-b border-outline-variant bg-surface-container-low">
            <span className="font-code-md text-code-md text-on-surface-variant/60">editor</span>
            <span className="font-code-md text-code-md text-on-surface-variant/60">
              {lines.length} lines · {value.length} chars
            </span>
          </div>
          <div className="flex max-h-[420px] overflow-hidden">
            <div className="w-10 shrink-0 text-right pr-2 py-2 font-code-md text-code-md text-on-surface-variant/30 select-none">
              {lines.map((_, i) => (
                <div key={i}>{i + 1}</div>
              ))}
            </div>
            <textarea
              value={value}
              onChange={(e) => onChange(e.target.value)}
              spellCheck={false}
              className="flex-1 bg-transparent p-2 font-code-md text-code-md text-on-surface resize-none focus:outline-none whitespace-pre overflow-auto sidebar-scroll h-[420px]"
              placeholder={"services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"80:80\""}
            />
          </div>
        </div>
      </div>

      <div className="space-y-md">
        <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md">
          <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Validation</p>
          {validation.isLoading && <div className="space-y-sm" aria-label="Validating compose file"><Skeleton variant="text" /><Skeleton variant="text" className="w-2/3" /></div>}
          {validation.data && validation.data.valid && (
            <div className="space-y-sm">
              <p className="font-body-sm text-body-sm text-[#4ade80] flex items-center gap-sm">
                <CheckCircle size={16} aria-hidden="true" />
                Compose file is valid
              </p>
              <div className="space-y-sm">
                {(validation.data.services ?? []).map((svc) => (
                  <div key={svc.name} className="flex items-center justify-between gap-sm">
                    <span className="font-code-md text-code-md text-on-surface">{svc.name}</span>
                    <span className="font-code-md text-code-md text-on-surface-variant/60 truncate max-w-[180px]">{svc.image || (svc.build ? "build:" + svc.build.slice(0, 40) : "—")}</span>
                  </div>
                ))}
              </div>
              {validation.data.total_ports > 0 && (
                <p className="font-code-md text-code-md text-on-surface-variant/60">{validation.data.total_ports} exposed port(s)</p>
              )}
            </div>
          )}
          {validation.data && !validation.data.valid && (
            <div className="space-y-sm">
              {(validation.data.errors ?? []).map((e, i) => (
                <p key={i} className="font-body-sm text-body-sm text-error flex items-start gap-sm">
                  <XCircle size={16} className="shrink-0" aria-hidden="true" />
                  {e}
                </p>
              ))}
            </div>
          )}
          {validation.data && validation.data.valid && (validation.data.warnings ?? []).length > 0 && (
            <div className="space-y-sm mt-sm border-t border-outline-variant pt-sm">
              {validation.data.warnings.map((w, i) => (
                <p key={i} className="font-body-sm text-body-sm text-[#fbbf24] flex items-start gap-sm">
                  <Warning size={16} className="shrink-0" aria-hidden="true" />
                  {w}
                </p>
              ))}
            </div>
          )}
        </div>
        <div className="bg-surface-container-low border border-outline-variant rounded-lg p-md">
          <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Service graph</p>
          {(validation.data?.services ?? []).map((svc) => (
            <div key={svc.name} className="flex items-center gap-sm font-code-md text-code-md py-1">
              <Database size={14} className="text-primary" aria-hidden="true" />
              <span className="text-on-surface">{svc.name}</span>
              {svc.image && <span className="text-on-surface-variant/60 truncate max-w-[140px]">← {svc.image}</span>}
            </div>
          ))}
          {(validation.data?.services ?? []).length === 0 && <p className="font-body-sm text-body-sm text-on-surface-variant">Dependencies appear here as you type.</p>}
        </div>
      </div>
    </div>
  );
}
