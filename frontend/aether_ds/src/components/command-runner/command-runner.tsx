import { Copy, Play, Stop } from "@phosphor-icons/react";
import { type ReactNode, useState } from "react";
export interface CommandRunnerProps {
  command?: string;
  target?: ReactNode;
  output?: string;
  status?: "idle" | "running" | "success" | "error" | "timeout" | "cancelled";
  onRun?: (command: string) => void;
  onCancel?: () => void;
  onRetry?: () => void;
  onCopy?: () => void;
  permissionDenied?: boolean;
}
export function CommandRunner({
  command: initialCommand = "",
  onCancel,
  onCopy,
  onRetry,
  onRun,
  output,
  permissionDenied,
  status = "idle",
  target,
}: CommandRunnerProps) {
  const [command, setCommand] = useState(initialCommand);
  return (
    <section className="overflow-hidden rounded-2xl border border-border bg-surface-card shadow-sm">
      <header className="flex flex-wrap items-center gap-2 border-b border-border bg-surface-container-low/70 p-4">
        <input
          value={command}
          onChange={(event) => setCommand(event.target.value)}
          aria-label="Command"
          placeholder="Enter command"
          className="h-10 min-w-48 flex-1 rounded-xl border border-border bg-surface-control px-3 font-mono text-code-md outline-none transition-[border-color,box-shadow,background-color] duration-200 hover:bg-surface-container-highest/40 focus:border-primary focus:bg-surface-card focus:ring-2 focus:ring-primary/20"
        />
        {target ? (
          <span className="text-body-sm text-muted-foreground">{target}</span>
        ) : null}
        <button
          type="button"
          disabled={permissionDenied || status === "running" || !command}
          onClick={() => onRun?.(command)}
          className="inline-flex items-center gap-1 rounded-xl bg-primary px-3 py-2 text-body-sm font-semibold text-primary-foreground outline-none transition-[background-color,box-shadow,transform] duration-150 hover:bg-primary-fixed focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] disabled:opacity-50"
        >
          <Play size={16} />
          Run
        </button>
        {status === "running" ? (
          <button
            type="button"
            onClick={onCancel}
            className="inline-flex items-center gap-1 rounded-xl border border-border px-3 py-2 text-body-sm outline-none transition-[background-color,box-shadow,transform] duration-150 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
          >
            <Stop size={16} />
            Cancel
          </button>
        ) : null}
      </header>
      {permissionDenied ? (
        <p className="border-b border-status-danger/20 bg-status-danger-container/10 p-4 text-body-sm text-status-danger">
          You do not have permission to run commands on this target.
        </p>
      ) : null}
      <pre className="min-h-32 overflow-auto bg-surface-lowest p-4 font-mono text-code-md text-foreground">
        {output ??
          (status === "running" ? "Running..." : "Output will appear here.")}
      </pre>
      <footer className="flex justify-end gap-3 border-t border-border bg-surface-container-low/50 p-4">
        {status === "error" || status === "timeout" ? (
          <button
            type="button"
            onClick={onRetry}
            className="rounded-lg px-2 py-1 text-body-sm font-semibold text-primary outline-none transition-[background-color,transform] duration-150 hover:bg-primary/10 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
          >
            Retry
          </button>
        ) : null}
        <button
          type="button"
          onClick={onCopy}
          className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-body-sm text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
        >
          <Copy size={16} />
          Copy output
        </button>
      </footer>
    </section>
  );
}
