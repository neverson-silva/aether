import React from "react";
import { cn } from "./cn";

export function AppPage({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn("max-w-[1600px] mx-auto", className)}>{children}</div>;
}

export function AppPageHeader({
  title,
  description,
  actions,
  className,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("mb-lg flex flex-col md:flex-row justify-between items-start md:items-end gap-md", className)}>
      <div className="min-w-0">
        <AppPageTitle>{title}</AppPageTitle>
        {description && <AppDescription>{description}</AppDescription>}
      </div>
      {actions && <div className="flex items-center gap-sm shrink-0">{actions}</div>}
    </div>
  );
}

export function AppPageTitle({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <h1 className={cn("font-headline-sm text-headline-sm font-bold text-on-surface tracking-tight mb-1", className)}>
      {children}
    </h1>
  );
}

export function AppDescription({ children, className }: { children: React.ReactNode; className?: string }) {
  return <p className={cn("font-body-sm text-body-sm text-on-surface-variant", className)}>{children}</p>;
}

export function AppSection({ title, actions, children, className }: { title?: React.ReactNode; actions?: React.ReactNode; children: React.ReactNode; className?: string }) {
  return (
    <section className={cn("mb-lg", className)}>
      {(title || actions) && <AppSectionHeader title={title} actions={actions} />}
      {children}
    </section>
  );
}

export function AppSectionHeader({ title, actions }: { title?: React.ReactNode; actions?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between mb-md">
      {title ? <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase tracking-wider">{title}</h2> : <span />}
      {actions}
    </div>
  );
}

export function AppToolbar({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn("flex items-center gap-sm flex-wrap", className)}>{children}</div>;
}

export function AppToolbarActions({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-sm ml-auto">{children}</div>;
}
