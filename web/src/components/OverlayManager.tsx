import React, { useEffect, useRef, useState } from "react";
import { useOverlayStore } from "../stores/overlay";

export function useOverlay(id: string) {
  const activeId = useOverlayStore((s) => s.activeId);
  return {
    active: activeId === id,
    open: () => useOverlayStore.getState().requestOpen(id),
    close: () => useOverlayStore.getState().requestClose(id),
  };
}

const OVERLAY_FADE_MS = 180;

export function useOverlayGate(id: string, open: boolean, onClosed?: () => void) {
  const { active, open: requestOpen, close: requestClose } = useOverlay(id);
  const [mounted, setMounted] = useState(open && active);
  const [closing, setClosing] = useState(false);
  const onClosedRef = useRef(onClosed);
  onClosedRef.current = onClosed;

  useEffect(() => {
    if (open) {
      requestOpen();
    } else if (active) {
      requestClose();
    }
  }, [open]);

  useEffect(() => {
    if (open && active) {
      setMounted(true);
      setClosing(false);
    } else if (mounted) {
      setClosing(true);
      const t = setTimeout(() => {
        setMounted(false);
        onClosedRef.current?.();
      }, OVERLAY_FADE_MS);
      return () => clearTimeout(t);
    }
  }, [open, active, mounted]);

  return { mounted, closing, close: () => requestClose() };
}

export function OverlayProvider({ children }: { children: React.ReactNode }) {
  const activeId = useOverlayStore((s) => s.activeId);
  const restoreFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (activeId) {
      restoreFocusRef.current = document.activeElement as HTMLElement | null;
    }
  }, [activeId]);

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = activeId ? "hidden" : "";
    return () => {
      document.body.style.overflow = prev;
    };
  }, [activeId]);

  useEffect(() => {
    if (!activeId) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        useOverlayStore.getState().requestClose(activeId);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [activeId]);

  useEffect(() => {
    if (!activeId) {
      restoreFocusRef.current?.focus?.();
      restoreFocusRef.current = null;
    }
  }, [activeId]);

  return <>{children}</>;
}