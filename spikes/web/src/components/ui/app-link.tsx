import React from "react";
import { Link } from "@tanstack/react-router";
import { cn } from "./cn";
import { BTN_BASE, BTN_VARIANTS, BTN_SIZES, type ButtonVariant, type ButtonSize } from "./button";

export function AppLink({
  to,
  href,
  external,
  variant = "default",
  size = "md",
  className,
  leftIcon,
  rightIcon,
  children,
}: {
  to?: string;
  href?: string;
  external?: boolean;
  variant?: ButtonVariant | "default";
  size?: ButtonSize;
  className?: string;
  leftIcon?: string;
  rightIcon?: string;
  children: React.ReactNode;
}) {
  const styles = cn(
    variant === "default" && "text-primary hover:underline",
    variant !== "default" && cn(BTN_BASE, BTN_VARIANTS[variant], BTN_SIZES[size])
  );
  const content = (
    <>
      {leftIcon && <span className="material-symbols-outlined text-[1.1em]" style={{ fontVariationSettings: "'FILL' 1" }}>{leftIcon}</span>}
      {children}
      {rightIcon && <span className="material-symbols-outlined text-[1.1em]">{rightIcon}</span>}
    </>
  );
  if (href || external) {
    return (
      <a href={href} target={external ? "_blank" : undefined} rel={external ? "noreferrer" : undefined} className={cn(styles, className)}>
        {content}
      </a>
    );
  }
  return (
    <Link to={to as never} className={cn(styles, className)}>
      {content}
    </Link>
  );
}
