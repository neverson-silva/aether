import React from "react";
import { cn, SpinnerMini } from "./cn";

export type ButtonVariant = "primary" | "ghost" | "danger" | "subtle" | "secondary" | "outline" | "success" | "warning" | "link" | "icon";
export type ButtonSize = "xs" | "sm" | "md" | "lg";

export const BTN_VARIANTS: Record<ButtonVariant, string> = {
  primary: "bg-[#4d7cfe] hover:bg-[#3d6cf5] text-white border border-transparent",
  secondary: "bg-secondary-container/20 hover:bg-secondary-container/40 text-on-surface border border-secondary/20",
  outline: "bg-transparent hover:bg-surface-container-high text-on-surface border border-outline-variant hover:border-primary",
  ghost: "bg-transparent hover:bg-surface-container-high text-on-surface border border-transparent",
  danger: "bg-[#e5484d] hover:bg-[#d03f44] text-white border border-transparent",
  success: "bg-[#10b981] hover:bg-[#059669] text-white border border-transparent",
  warning: "bg-[#f59e0b] hover:bg-[#d97706] text-white border border-transparent",
  link: "bg-transparent text-primary hover:underline border border-transparent p-0 h-auto",
  icon: "bg-transparent hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface border border-transparent",
  subtle: "bg-surface-container hover:bg-surface-container-high text-on-surface-variant border border-outline-variant",
};

export const BTN_SIZES: Record<ButtonSize, string> = {
  xs: "h-6 px-2 text-[11px] gap-1",
  sm: "h-8 px-sm text-label-caps gap-1.5",
  md: "h-9 px-md text-label-caps gap-2",
  lg: "h-11 px-lg text-label-caps gap-2",
};

export const BTN_BASE =
  "inline-flex items-center justify-center font-label-caps rounded-DEFAULT transition-all active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed select-none whitespace-nowrap focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 cursor-pointer";

export function Button({
  variant = "primary",
  size = "md",
  loading = false,
  leftIcon,
  rightIcon,
  className,
  children,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  leftIcon?: string;
  rightIcon?: string;
}) {
  return (
    <button
      className={cn(BTN_BASE, BTN_VARIANTS[variant], BTN_SIZES[size], className)}
      disabled={props.disabled || loading}
      {...props}
    >
      {loading ? (
        <SpinnerMini />
      ) : leftIcon ? (
        <span className="material-symbols-outlined text-[1.1em]" style={{ fontVariationSettings: "'FILL' 1" }}>
          {leftIcon}
        </span>
      ) : null}
      {children}
      {rightIcon && !loading && <span className="material-symbols-outlined text-[1.1em]">{rightIcon}</span>}
    </button>
  );
}

/* Convenção App* — aliases do mesmo botão unificado. */
export type AppButtonVariant = ButtonVariant;
export type AppButtonSize = ButtonSize;
export const AppButton = Button;
