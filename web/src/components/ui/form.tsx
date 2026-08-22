import React, { useEffect, useMemo, useRef, useState } from "react";
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
  const rootRef = useRef<HTMLDivElement>(null);
  const nativeRef = useRef<HTMLSelectElement | null>(null);
  const [open, setOpen] = useState(false);
  const optionElements = React.Children.toArray(children).filter(
    (child): child is React.ReactElement<React.OptionHTMLAttributes<HTMLOptionElement>> => React.isValidElement(child)
  );
  const options = useMemo(
    () =>
      optionElements.map((child) => ({
          disabled: Boolean(child.props.disabled),
          label: String(child.props.children ?? child.props.value ?? ""),
          value: String(child.props.value ?? ""),
        })),
    [optionElements]
  );
  const controlledValue = props.value == null ? undefined : String(props.value);
  const [selectedValue, setSelectedValue] = useState(() => controlledValue ?? String(props.defaultValue ?? options[0]?.value ?? ""));
  const value = controlledValue ?? selectedValue;
  const selectedOption = options.find((option) => option.value === value) ?? options[0];

  useEffect(() => {
    if (controlledValue !== undefined) setSelectedValue(controlledValue);
  }, [controlledValue]);

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  const selectValue = (nextValue: string) => {
    setSelectedValue(nextValue);
    setOpen(false);
    if (nativeRef.current) {
      nativeRef.current.value = nextValue;
      props.onChange?.({ target: nativeRef.current, currentTarget: nativeRef.current } as React.ChangeEvent<HTMLSelectElement>);
    }
  };

  return (
    <div ref={rootRef} className="relative w-full">
      <select
        ref={(node) => {
          nativeRef.current = node;
          if (typeof ref === "function") ref(node);
          else if (ref) (ref as React.MutableRefObject<HTMLSelectElement | null>).current = node;
        }}
        className="sr-only"
        {...props}
      >
        {children}
      </select>
      <button
        type="button"
        disabled={props.disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
        className={cn(
          "flex w-full items-center justify-between gap-sm bg-surface border border-outline-variant rounded-DEFAULT py-2 pl-sm pr-sm font-body-md text-body-md text-on-surface text-left focus:border-primary focus:ring-1 focus:ring-primary focus:outline-none transition-all cursor-pointer disabled:cursor-not-allowed disabled:opacity-50",
          className
        )}
      >
        <span className="min-w-0 truncate">{selectedOption?.label ?? "Select an option"}</span>
        <span className={cn("material-symbols-outlined shrink-0 text-on-surface-variant text-[18px] transition-transform", open && "rotate-180")}>expand_more</span>
      </button>
      {open && (
        <div role="listbox" className="absolute left-0 right-0 top-[calc(100%+4px)] z-30 max-h-64 overflow-y-auto rounded-DEFAULT border border-outline-variant bg-surface-container-high p-xs shadow-lg sidebar-scroll">
          {options.map((option) => (
            <button
              key={option.value}
              type="button"
              role="option"
              aria-selected={option.value === value}
              disabled={option.disabled}
              onClick={() => selectValue(option.value)}
              className={cn(
                "flex w-full items-center rounded-DEFAULT px-sm py-2 font-body-md text-body-md text-on-surface text-left transition-colors hover:bg-surface-container-highest hover:text-primary disabled:cursor-not-allowed disabled:opacity-50",
                option.value === value && "bg-primary/15 text-primary"
              )}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
});

export function TimePicker({ value, onChange, className }: { value: string; onChange: (value: string) => void; className?: string }) {
  const rootRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [hours, minutes] = value.split(":");
  const hourOptions = Array.from({ length: 24 }, (_, index) => String(index).padStart(2, "0"));
  const minuteOptions = Array.from({ length: 60 }, (_, index) => String(index).padStart(2, "0"));

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  return (
    <div ref={rootRef} className={cn("relative w-full", className)}>
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
        className="flex w-full items-center justify-between gap-sm bg-surface border border-outline-variant rounded-DEFAULT py-2 px-sm font-body-md text-body-md text-on-surface text-left focus:border-primary focus:ring-1 focus:ring-primary focus:outline-none transition-all"
      >
        <span>{value}</span>
        <span className={cn("material-symbols-outlined text-on-surface-variant text-[18px] transition-transform", open && "rotate-180")}>schedule</span>
      </button>
      {open && (
        <div className="absolute left-0 top-[calc(100%+4px)] z-30 grid w-40 grid-cols-2 gap-xs rounded-DEFAULT border border-outline-variant bg-surface-container-high p-xs shadow-lg">
          <div className="max-h-56 overflow-y-auto sidebar-scroll" role="listbox" aria-label="Hours">
            {hourOptions.map((hour) => (
              <button
                key={hour}
                type="button"
                onClick={() => onChange(`${hour}:${minutes || "00"}`)}
                className={cn("w-full rounded-DEFAULT px-sm py-1.5 font-code-md text-code-md text-on-surface transition-colors hover:bg-surface-container-highest", hour === hours && "bg-primary text-on-primary")}
              >
                {hour}
              </button>
            ))}
          </div>
          <div className="max-h-56 overflow-y-auto sidebar-scroll" role="listbox" aria-label="Minutes">
            {minuteOptions.map((minute) => (
              <button
                key={minute}
                type="button"
                onClick={() => onChange(`${hours || "00"}:${minute}`)}
                className={cn("w-full rounded-DEFAULT px-sm py-1.5 font-code-md text-code-md text-on-surface transition-colors hover:bg-surface-container-highest", minute === minutes && "bg-primary text-on-primary")}
              >
                {minute}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

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
