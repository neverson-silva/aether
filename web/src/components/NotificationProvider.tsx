import React, { useEffect, useRef } from "react";
import type { EventEnvelope, NotificationItem } from "../hooks";
import { useRealtimeEvent } from "./RealtimeProvider";
import { useToast, type ToastLevel } from "./ui";
import { useNotificationsStore } from "../stores/notifications";

interface NotificationContextValue {
  unread: number;
  openBell: () => void;
  list: NotificationItem[];
  markRead: (id: string) => void;
  markAllRead: () => void;
  refresh: () => void;
}



export function useNotifications(): NotificationContextValue {
  const unread = useNotificationsStore((st) => st.unread);
  const list = useNotificationsStore((st) => st.list);
  const openBell = () => useNotificationsStore.getState().toggleBell();
  const markRead = (id: string) => void useNotificationsStore.getState().markRead(id);
  const markAllRead = () => void useNotificationsStore.getState().markAllRead();
  const refresh = () => void useNotificationsStore.getState().refresh();
  return { unread, openBell, list, markRead, markAllRead, refresh };
}

function levelForType(type: string): ToastLevel {
  if (type.includes("failed") || type.includes("error")) return "error";
  if (type.includes("ready") || type.includes("finished") || type.includes("recovered")) return "success";
  if (type.includes("queued") || type.includes("building") || type.includes("starting") || type.includes("healthcheck")) return "warning";
  return "info";
}

function isNotifiable(type: string): boolean {
  if (type.startsWith("deploy.")) {
    return ["deploy.queued", "deploy.ready", "deploy.failed", "deploy.rolled_back", "deploy.cancelled"].includes(type);
  }
  return (
    type.startsWith("backup.") ||
    type.startsWith("server.") ||
    type.startsWith("database.") ||
    type.startsWith("domain.") ||
    type.startsWith("alert.") ||
    type.startsWith("member.") ||
    type.startsWith("env.")
  );
}

function toItem(ev: EventEnvelope): NotificationItem {
  return {
    id: ev.id,
    org_id: ev.org_id || "",
    type: ev.type,
    message: ev.message || ev.type,
    payload: JSON.stringify(ev.payload ?? {}),
    read: false,
    created_at: ev.ts,
  };
}

export function NotificationProvider({ children }: { children: React.ReactNode }) {
  const { toast } = useToast();

  useRealtimeEvent((ev, replay) => {
    if (!isNotifiable(ev.type)) return;
    if (replay) {
      useNotificationsStore.getState().prependReplay(toItem(ev));
      return;
    }
    useNotificationsStore.getState().prepend(toItem(ev));
    const level = levelForType(ev.type);
    const isError = level === "error";
    const target = (ev.payload?.service_id || ev.payload?.app_id) as string | undefined;
    if (ev.message) toast(ev.message, { level, onClick: isError ? () => window.location.assign(`/apps`) : undefined });
    if (target) {
      useNotificationsStore.getState().patchPayload(ev.id, target);
    }
  });

  useEffect(() => {
    void useNotificationsStore.getState().refresh();
    const onOrgChange = () => void useNotificationsStore.getState().refresh();
    window.addEventListener("aether:org", onOrgChange);
    const reconcile = setInterval(() => {
      void useNotificationsStore.getState().refresh();
    }, 30000);
    return () => {
      clearInterval(reconcile);
      window.removeEventListener("aether:org", onOrgChange);
    };
  }, []);

  return (
    <>
      {children}
      <BellHost />
    </>
  );
}

function BellHost() {
  const open = useNotificationsStore((st) => st.bellOpen);
  if (!open) return null;
  return <BellDropdown />;
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
                <span className={`font-body-sm text-body-sm line-clamp-5 break-words ${!n.read ? "text-on-surface font-semibold" : "text-on-surface-variant"}`} title={n.message}>{n.message}</span>
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
