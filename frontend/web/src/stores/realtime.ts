import { create } from "zustand";

interface RealtimeState {
  connected: boolean;
  lastSeq: number;
  setConnected: (v: boolean) => void;
  setLastSeq: (n: number) => void;
}

export const useRealtimeStore = create<RealtimeState>((set) => ({
  connected: false,
  lastSeq: 0,
  setConnected: (v) => set({ connected: v }),
  setLastSeq: (n) => set({ lastSeq: n }),
}));