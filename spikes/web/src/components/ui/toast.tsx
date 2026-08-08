import React, { createContext, useCallback, useContext } from "react";
import { Toaster, toast as sonnerToast } from "sonner";

export type ToastLevel = "success" | "error" | "info" | "warning";

interface ToastOptions {
  level?: ToastLevel;
  duration?: number;
  onClick?: () => void;
}

type ToastFn = (msg: string, opts?: ToastOptions | ToastLevel) => void;

const ToastCtx = createContext<{ toast: ToastFn }>({ toast: () => {} });

export function useToast() {
  return useContext(ToastCtx);
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const toast = useCallback<ToastFn>((message, optsOrLevel = {}) => {
    const opts: ToastOptions = typeof optsOrLevel === "string" ? { level: optsOrLevel } : optsOrLevel;
    const level = opts.level ?? "success";
    const duration = opts.duration ?? (level === "error" ? 10000 : level === "success" ? 8000 : level === "warning" ? 6000 : 5000);
    const base = {
      duration,
      closeButton: true,
      ...(opts.onClick
        ? {
            action: {
              label: "View",
              onClick: opts.onClick,
            },
          }
        : {}),
    };
    if (level === "error") sonnerToast.error(message, base);
    else if (level === "warning") sonnerToast.warning(message, base);
    else if (level === "info") sonnerToast.info(message, base);
    else sonnerToast.success(message, base);
  }, []);
  return (
    <ToastCtx.Provider value={{ toast }}>
      {children}
      <Toaster
        theme="dark"
        position="bottom-right"
        richColors
        gap={10}
        offset={20}
        toastOptions={{
          style: {
            background: "#201f1f",
            border: "1px solid rgba(255,255,255,0.08)",
            color: "#e5e2e1",
            fontFamily: "inherit",
            fontSize: "14px",
            borderRadius: "12px",
            boxShadow: "0 8px 32px rgba(0,0,0,0.45)",
          },
        }}
      />
    </ToastCtx.Provider>
  );
}
