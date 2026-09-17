import type { ReactNode } from "react";

export function PageHeader({
  actions,
  description,
  eyebrow,
  meta,
  title,
}: {
  actions?: ReactNode;
  description: string;
  eyebrow: string;
  meta?: ReactNode;
  title: string;
}) {
  return (
    <header className="relative isolate overflow-hidden rounded-2xl border border-border bg-gradient-to-br from-primary/10 via-surface-card to-secondary/5 px-lg py-xl shadow-sm">
      <div className="pointer-events-none absolute -right-16 -top-20 -z-10 size-56 rounded-full bg-primary/10 blur-3xl" aria-hidden="true" />
      <div className="pointer-events-none absolute -bottom-24 left-1/3 -z-10 size-48 rounded-full bg-secondary/10 blur-3xl" aria-hidden="true" />
      <div className="flex flex-col justify-between gap-lg sm:flex-row sm:items-end">
        <div className="min-w-0">
          <p className="font-label-caps text-label-caps uppercase text-primary">{eyebrow}</p>
          <h1 className="mt-sm font-display-lg text-display-lg tracking-[-0.04em] text-on-surface">{title}</h1>
          <p className="mt-sm max-w-2xl font-body-md text-body-md text-on-surface-variant">{description}</p>
          {meta ? <div className="mt-md flex flex-wrap items-center gap-sm">{meta}</div> : null}
        </div>
        {actions ? <div className="flex shrink-0 flex-wrap items-center gap-sm">{actions}</div> : null}
      </div>
    </header>
  );
}
