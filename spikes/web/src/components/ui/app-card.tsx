import React from "react";
import { cn } from "./cn";

export function AppCard({
  className,
  children,
  onClick,
  hover,
}: {
  className?: string;
  children: React.ReactNode;
  onClick?: () => void;
  hover?: boolean;
}) {
  return (
    <div
      onClick={onClick}
      className={cn(
        "bg-surface-container-low border border-outline-variant rounded-xl",
        onClick || hover ? "transition-all hover:border-primary/40 hover:shadow-lg" : "",
        className
      )}
    >
      {children}
    </div>
  );
}
