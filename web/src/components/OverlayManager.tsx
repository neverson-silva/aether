import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";

interface OverlayManagerCtx {
  activeId: string | null;
  requestOpen: (id: string) => void;
  requestClose: (id: string) => void;
}

const Ctx = createContext<OverlayManagerCtx>({
  activeId: null,
  requestOpen: () => {},
  requestClose: () => {},
});

export function OverlayProvider({ children }: { children: React.ReactNode }) {
  const [activeId, setActiveId] = useState<string | null>(null);
  const restoreFocusRef = useRef<HTMLElement | null>(null);

  const requestOpen = useCallback((id: string) => {
    setActiveId((prev) => {
      if (prev !== null && prev !== id) return prev;
      if (prev === null) {
        restoreFocusRef.current = document.activeElement as HTMLElement | null;
      }
      return id;
    });
  }, []);

  const requestClose = useCallback((id: string) => {
    setActiveId((prev) => {
      if (prev === id) return null;
      return prev;
    });
  }, []);

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = activeId ? "hidden" : "";
    return () => {
      document.body.style.overflow = prev;
    };
  }, [activeId]);

  useEffect(() => {
    if (activeId) {
      const onKey = (e: KeyboardEvent) => {
        if (e.key === "Escape") {
          e.preventDefault();
          setActiveId(null);
        }
      };
      window.addEventListener("keydown", onKey);
      return () => window.removeEventListener("keydown", onKey);
    }
    restoreFocusRef.current?.focus?.();
    restoreFocusRef.current = null;
  }, [activeId]);

  return <Ctx.Provider value={{ activeId, requestOpen, requestClose }}>{children}</Ctx.Provider>;
}

export function useOverlay(id: string) {
  const { activeId, requestOpen, requestClose } = useContext(Ctx);
  return {
    active: activeId === id,
    open: () => requestOpen(id),
    close: () => requestClose(id),
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

  return { mounted, closing, active, close: requestClose };
}
