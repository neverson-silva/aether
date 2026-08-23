import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useRealtimeStore } from "../stores/realtime";
import { ApiError, apiGet, clearToken, getServer, isPublicRoute } from "../api/client";
import type { EventEnvelope, RealtimeInbound } from "../hooks/types";

interface RealtimeContextValue {
  connected: boolean;
  lastSeq: number;
  subscribe: (fn: (ev: EventEnvelope, replay: boolean) => void) => () => void;
  subscribePresence: (fn: (scope: string, count: number) => void) => () => void;
  send: (message: { op: string; scope?: string }) => void;
}

const RealtimeCtx = createContext<RealtimeContextValue>({
  connected: false,
  lastSeq: 0,
  subscribe: () => () => {},
  subscribePresence: () => () => {},
  send: () => {},
});

function useRealtimeCtx(): RealtimeContextValue {
  return useContext(RealtimeCtx);
}

export function useRealtime(): RealtimeContextValue {
  const context = useRealtimeCtx();
  const connected = useRealtimeStore((st) => st.connected);
  const lastSeq = useRealtimeStore((st) => st.lastSeq);
  return { connected, lastSeq, subscribe: context.subscribe, subscribePresence: context.subscribePresence, send: context.send };
}

export function useRealtimeEvent(fn: (ev: EventEnvelope, replay: boolean) => void) {
  const subscribe = useRealtimeCtx().subscribe;
  const ref = useRef(fn);
  ref.current = fn;
  useEffect(() => subscribe((ev, replay) => ref.current(ev, replay)), [subscribe]);
}

function resourceKeys(ev: EventEnvelope): unknown[][] {
  if (ev.type.startsWith("deploy.")) {
    const appId =
      ev.app_id ||
      (ev.payload?.service_id as string | undefined) ||
      (ev.payload?.app_id as string | undefined);
    if (!appId) return [["deployments"], ["system-summary"]];
    return [
      ["app", appId],
      ["deployments", appId],
      ["apps"],
      ["timeline", appId],
      ["stats", appId],
      ["app-compose", appId],
      ["system-summary"],
    ];
  }
  if (ev.type.startsWith("backup")) {
    const databaseID = ev.payload?.database_id as string | undefined;
    return databaseID
      ? [["database-backups", databaseID], ["database-backup-configs", databaseID], ["databases"], ["backups"]]
      : [["databases"], ["backups"]];
  }
  if (ev.resource_type === "database") {
    return [["databases"]];
  }
  if (ev.type.startsWith("server")) {
    return [["servers"]];
  }
  if (ev.type.startsWith("domain")) {
    return [["domains"]];
  }
  if (ev.resource_type === "project") {
    return [["projects"]];
  }
  return [];
}

function applyInvalidation(qc: QueryClient, ev: EventEnvelope) {
  for (const key of resourceKeys(ev)) {
    qc.invalidateQueries({ queryKey: key });
  }
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const listeners = useRef(new Set<(ev: EventEnvelope, replay: boolean) => void>());
  const presenceListeners = useRef(new Set<(scope: string, count: number) => void>());
  const attemptRef = useRef(0);
  const seqRef = useRef(0);
  const sessionRef = useRef(0);


  const seqKey = useCallback(() => {
    const org = localStorage.getItem("aether_org") || "";
    return `aether_rt_seq_${org}`;
  }, []);

  const handle = useCallback(
    (ev: EventEnvelope, replay: boolean) => {
      if (ev.seq > seqRef.current) seqRef.current = ev.seq;
      useRealtimeStore.getState().setLastSeq(seqRef.current);
      applyInvalidation(qc, ev);
      listeners.current.forEach((fn) => {
        try {
          fn(ev, replay);
        } catch {
        }
      });
    },
    [qc],
  );

  const connect = useCallback(() => {
    if (isPublicRoute()) return;
    const session = ++sessionRef.current;
    wsRef.current?.close();
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const base = getServer() || "";
    const seq = Number.parseInt(localStorage.getItem(seqKey()) || "0", 10) || 0;
    seqRef.current = seq;
    useRealtimeStore.getState().setLastSeq(seq);
    const ws = new WebSocket(`${proto}//${window.location.host}${base}/api/v1/ws/realtime`);
    const pingRef = { id: 0 as number | undefined };
    ws.onopen = () => {
      attemptRef.current = 0;
      useRealtimeStore.getState().setConnected(true);
      ws.send(JSON.stringify({ op: "subscribe", subs: ["org"], seq }));
      pingRef.id = window.setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ op: "ping" }));
      }, 25000);
    };
    ws.onmessage = (e: MessageEvent) => {
      let msg: RealtimeInbound;
      try {
        msg = JSON.parse(e.data as string);
      } catch {
        return;
      }
      if (msg.op === "event" && msg.ev) handle(msg.ev, !!msg.replay);
      if (msg.op === "presence" && msg.scope) {
        presenceListeners.current.forEach((fn) => fn(msg.scope!, msg.n ?? 0));
      }
    };
    ws.onerror = () => ws.close();
    ws.onclose = () => {
      if (pingRef.id !== undefined) window.clearInterval(pingRef.id);
      useRealtimeStore.getState().setConnected(false);
      const saved = Number.parseInt(localStorage.getItem(seqKey()) || "0", 10) || 0;
      if (seqRef.current > 0 && saved < seqRef.current) {
        localStorage.setItem(seqKey(), String(seqRef.current));
      }
      if (sessionRef.current !== session) return;
      attemptRef.current += 1;
      const delay = Math.min(30000, Math.pow(2, Math.min(attemptRef.current - 1, 5)) * 1000);
      setTimeout(() => connect(), delay);
      if (attemptRef.current >= 3 && !isPublicRoute()) {
        apiGet("/api/v1/me").catch((err: unknown) => {
          if (err instanceof ApiError && err.status === 401 && !isPublicRoute()) {
            clearToken();
            window.location.href = "/login";
          }
        });
      }
    };
    wsRef.current = ws;
  }, [handle, seqKey]);

  useEffect(() => {
    connect();
    const onOrgChange = () => connect();
    const onFocus = () => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) connect();
    };
    window.addEventListener("aether:org", onOrgChange);
    window.addEventListener("focus", onFocus);
    return () => {
      window.removeEventListener("aether:org", onOrgChange);
      window.removeEventListener("focus", onFocus);
      wsRef.current?.close();
    };
  }, [connect]);

  const subscribe = useCallback((fn: (ev: EventEnvelope, replay: boolean) => void) => {
    listeners.current.add(fn);
    return () => {
      listeners.current.delete(fn);
    };
  }, []);

  const subscribePresence = useCallback((fn: (scope: string, count: number) => void) => {
    presenceListeners.current.add(fn);
    return () => {
      presenceListeners.current.delete(fn);
    };
  }, []);

  const send = useCallback((message: { op: string; scope?: string }) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    }
  }, []);

  const value: RealtimeContextValue = { connected: useRealtimeStore.getState().connected, lastSeq: useRealtimeStore.getState().lastSeq, subscribe, subscribePresence, send };

  return <RealtimeCtx.Provider value={value}>{children}</RealtimeCtx.Provider>;
}
