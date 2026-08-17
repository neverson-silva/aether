import React from "react";
import { cn } from "./cn";

export const Input = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement> & { icon?: string }
>(function Input({ className, icon, ...props }, ref) {
  return (
    <div className="relative group w-full">
      {icon && (
        <span className="material-symbols-outlined absolute left-sm top-1/2 -translate-y-1/2 text-on-surface-variant group-focus-within:text-primary transition-colors text-[20px] pointer-events-none">
          {icon}
        </span>
      )}
      <input
        ref={ref}
        className={cn(
          "w-full bg-surface border border-outline-variant rounded-DEFAULT py-2 px-sm font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant/50 focus:border-primary focus:ring-1 focus:ring-primary focus:outline-none transition-all",
          icon && "pl-[36px]",
          className
        )}
        {...props}
      />
    </div>
  );
});

export const Select = React.forwardRef<
  HTMLSelectElement,
  React.SelectHTMLAttributes<HTMLSelectElement>
>(function Select({ className, children, ...props }, ref) {
  return (
    <div className="relative w-full">
      <select
        ref={ref}
        className={cn(
          "w-full appearance-none bg-surface border border-outline-variant rounded-DEFAULT py-2 pl-sm pr-9 font-body-md text-body-md text-on-surface focus:border-primary focus:ring-1 focus:ring-primary focus:outline-none transition-all cursor-pointer",
          className
        )}
        {...props}
      >
        {children}
      </select>
      <span className="material-symbols-outlined pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-on-surface-variant text-[18px]">
        expand_more
      </span>
    </div>
  );
});

export function Field({
  label,
  children,
  hint,
}: {
  label: string;
  children: React.ReactNode;
  hint?: string;
}) {
  return (
    <div className="space-y-sm">
      <label className="font-label-caps text-label-caps text-on-surface-variant block uppercase">
        {label}
      </label>
      {children}
      {hint && <p className="font-body-sm text-body-sm text-on-surface-variant/70">{hint}</p>}
    </div>
  );
}
