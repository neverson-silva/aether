import { CaretDown, CaretUp, Check, Copy } from "@phosphor-icons/react";
import { useState } from "react";
export interface CodeBlockProps {
  code: string;
  language?: string;
  title?: string;
  collapsible?: boolean;
  defaultExpanded?: boolean;
  onCopy?: () => void;
}
export function CodeBlock({
  code,
  collapsible,
  defaultExpanded = true,
  language,
  onCopy,
  title,
}: CodeBlockProps) {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(defaultExpanded);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      onCopy?.();
      window.setTimeout(() => setCopied(false), 1400);
    } catch {
      setCopied(false);
    }
  };
  return (
    <section className="overflow-hidden rounded-2xl border border-border bg-surface-lowest shadow-sm">
      <header className="flex items-center justify-between border-b border-border bg-surface-container-low/80 px-3 py-2">
        <span className="text-body-sm text-muted-foreground">
          {title ?? language ?? "Code"}
        </span>
        <div className="flex items-center gap-1">
          {collapsible ? (
            <button
              type="button"
              onClick={() => setExpanded(!expanded)}
              aria-label={expanded ? "Collapse code" : "Expand code"}
              className="rounded-lg p-1.5 text-muted-foreground outline-none transition-[background-color,transform] duration-150 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-95"
            >
              {expanded ? <CaretUp size={16} /> : <CaretDown size={16} />}
            </button>
          ) : null}
          <button
            type="button"
            onClick={copy}
            className="inline-flex items-center gap-1 rounded-lg p-1.5 text-body-sm text-muted-foreground outline-none transition-[background-color,transform] duration-150 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
          >
            {copied ? (
              <Check size={16} className="text-status-success" />
            ) : (
              <Copy size={16} />
            )}
            {copied ? "Copied" : "Copy"}
          </button>
        </div>
      </header>
      {expanded ? (
        <pre className="overflow-x-auto p-4 text-code-md text-foreground">
          <code>{code}</code>
        </pre>
      ) : null}
    </section>
  );
}
