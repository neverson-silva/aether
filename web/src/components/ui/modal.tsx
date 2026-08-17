import React, { useEffect } from "react";
import { Button } from "./button";
import { cn } from "./cn";

export function Modal({
  open,
  onClose,
  title,
  children,
  wide,
  size = "md",
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  wide?: boolean;
  size?: "sm" | "md" | "lg";
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);
  if (!open) return null;
  const maxW = wide ? "max-w-3xl" : size === "lg" ? "max-w-[36rem]" : size === "sm" ? "max-w-[28rem]" : "max-w-[32rem]";
  return (
    <div className="fixed inset-0 z-[80] flex items-center justify-center bg-black/60 p-4" onClick={onClose}>
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onClick={(e) => e.stopPropagation()}
        className={cn(
          "bg-surface-container-low border border-outline-variant rounded-xl shadow-[0_8px_32px_rgba(0,0,0,0.4)] w-full animate-modal-pop",
          maxW,
          "max-h-[90vh] overflow-y-auto sidebar-scroll"
        )}
      >
        <div className="flex items-center justify-between px-lg py-md border-b border-outline-variant sticky top-0 bg-surface-container-low z-10">
          <h2 className="font-headline-sm text-headline-sm font-semibold text-on-surface">{title}</h2>
          <button onClick={onClose} className="text-on-surface-variant hover:text-on-surface transition-colors">
            <span className="material-symbols-outlined">close</span>
          </button>
        </div>
        <div className="p-lg">{children}</div>
      </div>
    </div>
  );
}

export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  description,
  confirmLabel = "Confirm",
  danger,
  requireType,
}: {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: string;
  confirmLabel?: string;
  danger?: boolean;
  requireType?: string;
}) {
  const [typed, setTyped] = React.useState("");
  React.useEffect(() => {
    if (open) setTyped("");
  }, [open]);
  const matches = !requireType || typed === requireType;
  return (
    <Modal open={open} onClose={onClose} title={title} size="lg">
      <div className="py-sm">
        <p className="font-body-md text-body-md text-on-surface-variant leading-relaxed">{description}</p>
        {requireType && (
          <div className="mt-lg">
            <label className="font-label-caps text-label-caps text-on-surface-variant block mb-sm uppercase">
              Type <span className="font-code-md text-primary">{requireType}</span> to confirm
            </label>
            <input
              autoFocus
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              placeholder={`Type "${requireType}"`}
              className="w-full bg-surface border border-outline-variant rounded-DEFAULT py-2.5 px-sm font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant/50 focus:border-primary focus:ring-1 focus:ring-primary focus:outline-none transition-all"
            />
          </div>
        )}
      </div>
      <div className="flex justify-end gap-md border-t border-outline-variant mt-lg pt-lg">
        <Button variant="ghost" onClick={onClose}>
          Cancel
        </Button>
        <Button
          variant={danger ? "danger" : "primary"}
          disabled={!matches}
          onClick={() => {
            onConfirm();
            onClose();
          }}
        >
          {confirmLabel}
        </Button>
      </div>
    </Modal>
  );
}
