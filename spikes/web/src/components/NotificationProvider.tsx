import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { ApiError, apiGet, apiPost, clearToken, getServer, isPublicRoute } from "../api/client";
import type { NotificationItem } from "../api/hooks";
import { useToast, type ToastLevel } from "./ui";

function invalidateForEvent(qc: QueryClient, type: string, payload: { service_id?: string; app_id?: string; database?: string }) {
  const appId = payload?.service_id || payload?.app_id;
  if (appId) {
    qc.invalidateQueries({ queryKey: ["apps"] });
    qc.invalidateQueries({ queryKey: ["apps", appId] });
    qc.invalidateQueries({ queryKey: ["deployments", appId] });
    qc.invalidateQueries({ queryKey: ["stats", appId] });
    qc.invalidateQueries({ queryKey: ["timeline", appId] });
    qc.invalidateQueries({ queryKey: ["app-compose", appId] });
    qc.invalidateQueries({ queryKey: ["system-summary"] });
  }
  if (type.startsWith("backup") || payload?.database) {
    qc.invalidateQueries({ queryKey: ["databases"] });
    qc.invalidateQueries({ queryKey: ["backups"] });
  }
  if (type.startsWith("server")) {
    qc.invalidateQueries({ queryKey: ["servers"] });
  }
}

interface NotificationContextValue {
  unread: number;
  openBell: () => void;
  list: NotificationItem[];
  markRead: (id: string) => void;
  markAllRead: () => void;
  refresh: () => void;
}

const NotificationCtx = createContext<NotificationContextValue>({
  unread: 0,
  openBell: () => {},
  list: [],
  markRead: () => {},
  markAllRead: () => {},
  refresh: () => {},
});

export function useNotifications(): NotificationContextValue {
  return useContext(NotificationCtx);
}

function levelForType(type: string): ToastLevel {
  if (type.includes("failed") || type.includes("error")) return "error";
  if (type.includes("ready") || type.includes("finished") || type.includes("recovered")) return "success";
  if (type.includes("queued") || type.includes("building") || type.includes("starting") || type.includes("healthcheck")) return "warning";
  return "info";
}

export function NotificationProvider({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();
  const { toast } = useToast();
  const [unread, setUnread] = useState(0);
  const [list, setList] = useState<NotificationItem[]>([]);
  const [bellOpen, setBellOpen] = useState(false);
  const esRef = useRef<EventSource | null>(null);
  const attemptRef = useRef(0);

  const refresh = async () => {
    try {
      const [notifs, count] = await Promise.all([
        apiGet<NotificationItem[]>("/api/v1/notifications"),
        apiGet<{ count: number }>("/api/v1/notifications/unread-count"),
      ]);
      setList(notifs ?? []);
      setUnread(count.count);
    } catch {
      /* offline */
    }
  };

  const markRead = async (id: string) => {
    setList((prev) => prev.map((n) => (n.id === id ? { ...n, read: true } : n)));
    setUnread((u) => Math.max(0, u - 1));
    try {
      await apiPost(`/api/v1/notifications/${id}/read`);
      qc.invalidateQueries({ queryKey: ["notifications"] });
    } catch {
      /* ignore */
    }
  };

  const markAllRead = async () => {
    setList((prev) => prev.map((n) => ({ ...n, read: true })));
    setUnread(0);
    try {
      await apiPost("/api/v1/notifications/read-all");
      qc.invalidateQueries({ queryKey: ["notifications"] });
    } catch {
      /* ignore */
    }
  };

  const connect = () => {
    if (isPublicRoute()) return;
    esRef.current?.close();
    const base = getServer() || "";
    const org = localStorage.getItem("aether_org") || "";
    const es = new EventSource(`${base}/api/v1/events/stream${org ? "?org=" + encodeURIComponent(org) : ""}`, { withCredentials: true });
    // histórico (reconexão/offline→online): povoa a lista SEM toast
    es.addEventListener("history", (e: MessageEvent) => {
      const data = JSON.parse(e.data as string);
      setList((prev) => {
        const existing = new Set(prev.map((n) => n.id));
        if (existing.has(data.id)) return prev;
        return [{ id: data.id, org_id: "", type: data.type, message: data.message, payload: JSON.stringify(data.payload ?? {}), read: data.read, created_at: data.ts }, ...prev].slice(0, 100);
      });
    });
    es.addEventListener("notification", (e: MessageEvent) => {
      const data = JSON.parse(e.data as string);
      setList((prev) => {
        const next = [{ id: data.id, org_id: "", type: data.type, message: data.message, payload: JSON.stringify(data.payload ?? {}), read: data.read, created_at: data.ts }, ...prev];
        return next.slice(0, 100);
      });
      if (!data.read) setUnread((u) => u + 1);
      const level = levelForType(data.type);
      const isError = level === "error";
      const notif = data.payload as { service_id?: string; app_id?: string; database?: string };
      toast(data.message, { level, onClick: isError ? () => window.location.assign(`/apps`) : undefined });
      invalidateForEvent(qc, data.type, notif);
    });
    es.onopen = () => {
      attemptRef.current = 0;
    };
    es.onerror = () => {
      es.close();
      attemptRef.current += 1;
      if (attemptRef.current >= 3 && !isPublicRoute()) {
        apiGet("/api/v1/me").catch((e: unknown) => {
          if (e instanceof ApiError && e.status === 401 && !isPublicRoute()) {
            clearToken();
            window.location.href = "/login";
          }
        });
      }
      const delay = Math.min(30000, Math.pow(2, Math.min(attemptRef.current - 1, 5)) * 1000);
      setTimeout(connect, delay);
      // fallback: polling a cada 30s quando SSE falha muito
      setTimeout(refresh, 30000);
    };
    esRef.current = es;
  };

  useEffect(() => {
    refresh();
    connect();
    const onOrgChange = () => {
      refresh();
      connect();
    };
    window.addEventListener("aether:org", onOrgChange);
    const poll = setInterval(() => {
      if (!esRef.current || esRef.current.readyState === EventSource.CLOSED) {
        refresh();
      }
    }, 30000);
    return () => {
      clearInterval(poll);
      window.removeEventListener("aether:org", onOrgChange);
      esRef.current?.close();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const value: NotificationContextValue = {
    unread,
    openBell: () => setBellOpen((v) => !v),
    list,
    markRead,
    markAllRead,
    refresh,
  };

  return (
    <NotificationCtx.Provider value={value}>
      {children}
      {bellOpen && <BellDropdown />}
    </NotificationCtx.Provider>
  );
}

function BellDropdown() {
  const { list, markRead, markAllRead, openBell } = useNotifications();
  const ref = React.useRef<HTMLDivElement>(null);
  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) openBell();
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, [openBell]);
  return (
    <div ref={ref} className="fixed right-4 top-14 z-[70] w-[calc(100vw-32px)] max-w-96 rounded-xl bg-surface-popover border border-border-subtle shadow-md overflow-hidden">
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-border-subtle">
        <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Notifications</span>
        <button onClick={markAllRead} className="font-body-sm text-body-sm text-primary hover:underline">
          Mark all as read
        </button>
      </div>
      <div className="max-h-96 overflow-y-auto">
        {list.length === 0 && (
          <p className="font-body-sm text-body-sm text-on-surface-variant p-4 text-center">No notifications yet.</p>
        )}
        {list.slice(0, 50).map((n) => {
          let parsed: Record<string, string> = {};
          try {
            parsed = JSON.parse(n.payload || "{}");
          } catch {
            /* ignore */
          }
          const meta =
            n.type.includes("failed")
              ? { icon: "error", color: "text-error" }
              : n.type.includes("ready") || n.type.includes("finished")
                ? { icon: "check_circle", color: "text-[#4ade80]" }
                : n.type.includes("queued") || n.type.includes("building") || n.type.includes("starting") || n.type.includes("healthcheck")
                  ? { icon: "hourglass_top", color: "text-[#fbbf24]" }
                  : n.type.includes("server")
                    ? { icon: "dns", color: "text-[#60a5fa]" }
                    : n.type.includes("backup")
                      ? { icon: "backup", color: "text-[#4ade80]" }
                      : n.type.includes("alert")
                        ? { icon: "notifications_active", color: "text-error" }
                        : { icon: "notifications", color: "text-on-surface-variant" };
          const target = parsed.app_id || parsed.service_id;
          return (
            <button
              key={n.id}
              onClick={() => {
                markRead(n.id);
                if (target) window.location.assign(`/apps/${target}`);
              }}
              className={`w-full flex items-start gap-2 px-3 py-2.5 hover:bg-surface-container-high transition-colors text-left ${!n.read ? "bg-surface-container-high/40" : ""}`}
            >
              <span className={`material-symbols-outlined text-[16px] shrink-0 mt-0.5 ${meta.color}`}>{meta.icon}</span>
              <span className="flex-1 min-w-0">
                <span className={`block font-body-sm text-body-sm ${!n.read ? "text-on-surface font-semibold" : "text-on-surface-variant"}`}>{n.message}</span>
                <span className="block font-code-md text-code-md text-on-surface-variant/60">{relativeTime(n.created_at)}</span>
              </span>
              {!n.read && <span className="w-1.5 h-1.5 rounded-full bg-primary shrink-0 mt-1.5" />}
            </button>
          );
        })}
      </div>
      <button
        onClick={() => window.location.assign(`/notifications`)}
        className="w-full px-3 py-2 border-t border-border-subtle font-label-caps text-label-caps text-primary uppercase hover:bg-surface-container-high transition-colors"
      >
        View all history
      </button>
    </div>
  );
}

export function BellButton() {
  const { unread, openBell } = useNotifications();
  return (
    <button
      onClick={openBell}
      className="relative text-on-surface-variant hover:text-primary transition-colors flex items-center justify-center w-8 h-8 rounded-full hover:bg-surface-container-high"
      aria-label={`Notifications${unread ? ` (${unread} unread)` : ""}`}
    >
      <span className="material-symbols-outlined">notifications</span>
      {unread > 0 && (
        <span className="absolute -top-0.5 -right-0.5 min-w-4 h-4 px-1 rounded-full bg-error text-on-error font-label-caps text-[10px] flex items-center justify-center">
          {unread > 99 ? "99+" : unread}
        </span>
      )}
    </button>
  );
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const sec = Math.floor(diff / 1000);
  if (sec < 10) return "just now";
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const d = Math.floor(hr / 24);
  return `${d}d ago`;
}
