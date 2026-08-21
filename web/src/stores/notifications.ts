import { create } from "zustand";
import { apiGet, apiPost } from "../api/client";
import type { NotificationItem } from "../hooks";

interface NotificationsState {
  unread: number;
  list: NotificationItem[];
  bellOpen: boolean;
  setUnread: (n: number) => void;
  setList: (list: NotificationItem[]) => void;
  prepend: (item: NotificationItem) => void;
  prependReplay: (item: NotificationItem) => void;
  patchPayload: (id: string, appId: string) => void;
  refresh: () => Promise<void>;
  markRead: (id: string) => Promise<void>;
  markAllRead: () => Promise<void>;
  toggleBell: () => void;
  closeBell: () => void;
}

export const useNotificationsStore = create<NotificationsState>((set, get) => ({
  unread: 0,
  list: [],
  bellOpen: false,
  setUnread: (n) => set({ unread: n }),
  setList: (list) => set({ list }),
  prepend: (item) => {
    if (import.meta.env.DEV) console.warn("[bell] live event", item.type, "unread", get().unread, "->", get().unread + 1);
    return set((s) => ({
      list: [item, ...s.list.filter((n) => n.id !== item.id)].slice(0, 100),
      unread: s.unread + 1,
    }));
  },
  prependReplay: (item) =>
    set((s) => (s.list.some((n) => n.id === item.id) ? s : { list: [item, ...s.list].slice(0, 100) })),
  patchPayload: (id, appId) =>
    set((s) => ({
      list: s.list.map((n) =>
        n.id === id ? { ...n, payload: JSON.stringify({ ...(JSON.parse(n.payload || "{}") as object), app_id: appId }) } : n,
      ),
    })),
  refresh: async () => {
    try {
      const [notifs, count] = await Promise.all([
        apiGet<NotificationItem[]>("/api/v1/notifications"),
        apiGet<{ count: number }>("/api/v1/notifications/unread-count"),
      ]);
      get().setList(notifs ?? []);
      get().setUnread(count.count);
    } catch {
      /* offline */
    }
  },
  markRead: async (id) => {
    set((s) => ({
      list: s.list.map((n) => (n.id === id ? { ...n, read: true } : n)),
      unread: Math.max(0, s.unread - 1),
    }));
    try {
      await apiPost(`/api/v1/notifications/${id}/read`);
    } catch {
      /* ignore */
    }
  },
  markAllRead: async () => {
    set((s) => ({ list: s.list.map((n) => ({ ...n, read: true })), unread: 0 }));
    try {
      await apiPost("/api/v1/notifications/read-all");
    } catch {
      /* ignore */
    }
  },
  toggleBell: () => set((s) => ({ bellOpen: !s.bellOpen })),
  closeBell: () => set({ bellOpen: false }),
}));