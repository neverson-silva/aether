import { create } from "zustand";

interface OverlayState {
  activeId: string | null;
  requestOpen: (id: string) => void;
  requestClose: (id: string) => void;
}

export const useOverlayStore = create<OverlayState>((set) => ({
  activeId: null,
  requestOpen: (id) =>
    set((s) => {
      if (s.activeId !== null && s.activeId !== id) return s;
      return { activeId: id };
    }),
  requestClose: (id) => set((s) => (s.activeId === id ? { activeId: null } : s)),
}));