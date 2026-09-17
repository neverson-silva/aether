import { Pause, Play, WifiHigh, WifiSlash } from "@phosphor-icons/react";
import type { ReactNode } from "react";
import {
  ActivityFeed,
  type ActivityFeedProps,
} from "../activity-feed/activity-feed";
export interface RealtimeActivitySurfaceProps extends ActivityFeedProps {
  connected?: boolean;
  unreadCount?: number;
  onPause?: () => void;
  onResume?: () => void;
  onJumpToLatest?: () => void;
  paused?: boolean;
  connectionState?: "connected" | "reconnecting" | "offline";
  backfill?: ReactNode;
  conflict?: ReactNode;
}
export function RealtimeActivitySurface({
  connected = true,
  onJumpToLatest,
  onPause,
  onResume,
  unreadCount = 0,
  paused = false,
  connectionState,
  backfill,
  conflict,
  ...props
}: RealtimeActivitySurfaceProps) {
  return (
    <section className="space-y-3 rounded-2xl border border-border bg-surface-card p-4 shadow-sm">
      <header className="flex items-center justify-between">
        <span
          className={`inline-flex items-center gap-2 text-body-sm ${connected ? "text-status-success" : "text-status-danger"}`}
        >
          {connected ? <WifiHigh size={16} /> : <WifiSlash size={16} />}
          {connectionState === "reconnecting"
            ? "Reconnecting"
            : connectionState === "offline" || !connected
              ? "Disconnected"
              : "Connected"}
        </span>
        <div className="flex items-center gap-2">
          {unreadCount ? (
            <button
              type="button"
              onClick={onJumpToLatest}
              className="rounded-xl border border-primary/30 px-2.5 py-1.5 text-body-sm text-primary outline-none transition-[background-color,transform] duration-150 hover:bg-primary/10 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
            >
              {unreadCount} new
            </button>
          ) : null}
          {props.realtime ? (
            <button
              type="button"
              onClick={paused ? onResume : onPause}
              aria-label={paused ? "Resume activity" : "Pause activity"}
              className="rounded-lg p-1.5 text-muted-foreground outline-none transition-[background-color,transform] duration-150 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-95"
            >
              {paused ? <Play size={16} /> : <Pause size={16} />}
            </button>
          ) : null}
        </div>
      </header>
      {paused ? (
        <div className="rounded-xl border border-status-warning/30 bg-status-warning-container/20 px-3 py-2 text-body-sm text-status-warning-foreground">
          Activity is paused. New events are retained until resumed.
        </div>
      ) : null}
      {backfill ? <div>{backfill}</div> : null}
      {conflict ? <div>{conflict}</div> : null}
      <ActivityFeed {...props} realtime={connected} />
    </section>
  );
}
