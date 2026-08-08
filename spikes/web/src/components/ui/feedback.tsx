import React, { useCallback, useState } from "react";
import { cn } from "./cn";

export function Popover({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "absolute top-full mt-2 z-[70] min-w-60 rounded-xl bg-surface-popover text-popover-foreground border border-border-subtle shadow-md p-1",
        className
      )}
    >
      {children}
    </div>
  );
}

export function Spinner({ label }: { label?: string }) {
  return (
    <div className="flex items-center gap-2 text-on-surface-variant font-body-sm text-body-sm py-md justify-center">
      <span className="w-4 h-4 border-2 border-outline-variant border-t-primary rounded-full animate-spin" />
      {label || "Loading..."}
    </div>
  );
}

export function EmptyState({
  icon = "inbox",
  title,
  description,
  action,
}: {
  icon?: string;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-xl text-center">
      <span className="material-symbols-outlined text-[40px] text-on-surface-variant/40 mb-md">
        {icon}
      </span>
      <p className="font-headline-sm text-headline-sm text-on-surface mb-xs">{title}</p>
      {description && (
        <p className="font-body-sm text-body-sm text-on-surface-variant max-w-[24rem]">{description}</p>
      )}
      {action && <div className="mt-md">{action}</div>}
    </div>
  );
}

export function CodeBlock({ children }: { children: React.ReactNode }) {
  const [copied, setCopied] = useState(false);
  const copy = useCallback(async () => {
    await navigator.clipboard.writeText(String(children));
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [children]);
  return (
    <div className="relative">
      <pre className="bg-surface-container-lowest border border-outline-variant rounded-lg p-md font-code-md text-code-md text-on-surface overflow-auto max-h-[420px]">
        {children}
      </pre>
      <button
        onClick={copy}
        className="absolute top-2 right-2 flex items-center gap-1 font-label-caps text-label-caps text-on-surface-variant bg-surface-container-high border border-outline-variant rounded-DEFAULT px-2 py-1 hover:text-primary transition-colors"
      >
        <span className="material-symbols-outlined text-[14px]">
          {copied ? "check" : "content_copy"}
        </span>
        {copied ? "Copied" : "Copy"}
      </button>
    </div>
  );
}
