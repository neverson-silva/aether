import React from "react";
import { cn } from "./cn";
import { AppCard } from "./app-card";
import { SpinnerMini } from "./cn";

export function AppEmptyState({
  icon = "inbox",
  title,
  description,
  action,
  className,
}: {
  icon?: string;
  title: React.ReactNode;
  description?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <AppCard className={cn("p-xl text-center", className)}>
      <div className="w-12 h-12 mx-auto rounded-full bg-surface-container-high flex items-center justify-center mb-md">
        <span className="material-symbols-outlined text-on-surface-variant text-2xl">{icon}</span>
      </div>
      <h3 className="font-headline-sm text-headline-sm font-semibold text-on-surface mb-xs">{title}</h3>
      {description && <p className="font-body-sm text-body-sm text-on-surface-variant mb-md max-w-[24rem] mx-auto">{description}</p>}
      {action && <div className="flex justify-center">{action}</div>}
    </AppCard>
  );
}

export function AppErrorState({ message, onRetry, className }: { message?: string; onRetry?: () => void; className?: string }) {
  return (
    <AppEmptyState
      className={className}
      icon="error"
      title="Something went wrong"
      description={message}
      action={onRetry ? <button onClick={onRetry} className="px-md py-1.5 rounded bg-primary text-on-primary text-label-caps uppercase">Retry</button> : undefined}
    />
  );
}

export function AppLoading({ label = "Loading...", className }: { label?: string; className?: string }) {
  return (
    <div className={cn("flex items-center justify-center gap-sm py-lg", className)}>
      <SpinnerMini />
      <span className="font-body-sm text-body-sm text-on-surface-variant">{label}</span>
    </div>
  );
}

export function AppSkeleton({ className }: { className?: string }) {
  return <div className={cn("skeleton rounded-lg", className)} />;
}

export function AppSkeletonCard({ className }: { className?: string }) {
  return (
    <AppCard className={cn("p-md", className)}>
      <AppSkeleton className="h-6 w-6" />
      <AppSkeleton className="h-4 w-24 mt-md" />
      <AppSkeleton className="h-3 w-16 mt-sm" />
    </AppCard>
  );
}
