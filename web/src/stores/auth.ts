import { create } from "zustand";
import type { Me } from "../api/types";

interface AuthState {
  user: Me | null;
  isAuthenticated: boolean;
  setUser: (user: Me | null) => void;
  clear: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  setUser: (user) => set({ user, isAuthenticated: !!user }),
  clear: () => set({ user: null, isAuthenticated: false }),
}));