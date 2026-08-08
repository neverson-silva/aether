import React from "react";
import { cn } from "./cn";

export function Card({
  className,
  variant = "default",
  onClick,
  children,
}: {
  className?: string;
  variant?: "default" | "glass";
  onClick?: () => void;
  children: React.ReactNode;
}) {
  return (
    <div
      onClick={onClick}
      className={cn(
        variant === "glass"
          ? "glass-panel rounded-xl p-lg relative overflow-hidden group hover:border-primary/50 transition-colors"
          : "bg-surface-container-low border border-outline-variant rounded-lg p-md",
        className
      )}
    >
      {children}
    </div>
  );
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn("skeleton", className)} />;
}

export function SkeletonList({ rows = 3 }: { rows?: number }) {
  return (
    <div className="space-y-sm">
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} className="h-10" />
      ))}
    </div>
  );
}
