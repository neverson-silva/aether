import { useCallback, useEffect, useRef, useState } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import { Unicode11Addon } from "@xterm/addon-unicode11";
import "@xterm/xterm/css/xterm.css";
import { Card } from "../../../../components/ui";
import { getServer } from "../../../../api/client";

const SHELLS = [
  { id: "sh", label: "/bin/sh", cmd: "/bin/sh" },
  { id: "bash", label: "bash", cmd: "bash" },
  { id: "ash", label: "ash", cmd: "ash" },
];

const STATUS_META: Record<string, { dot: string; label: string }> = {
  connected: { dot: "bg-[#4ade80]", label: "Connected" },
  connecting: { dot: "bg-[#fbbf24] animate-pulse", label: "Connecting" },
  reconnecting: { dot: "bg-[#fbbf24] animate-pulse", label: "Reconnecting" },
  disconnected: { dot: "bg-error", label: "Disconnected" },
};

export function Terminal({ appID }: { appID: string }) {
  const hostRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<XTerm | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const searchRef = useRef<SearchAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const genRef = useRef(0);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [connState, setConnState] = useState<"connecting" | "connected" | "reconnecting" | "disconnected">("connecting");
  const [shell, setShell] = useState("sh");
  const [searchOpen, setSearchOpen] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);

  const sendResize = useCallback(() => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      const cols = termRef.current?.cols ?? 120;
      const rows = termRef.current?.rows ?? 30;
      ws.send(JSON.stringify({ type: "resize", cols, rows }));
    }
  }, []);

  const open = useCallback(
    (shellCmd: string) => {
      const gen = ++genRef.current;
      const host = hostRef.current;
      if (!host) return;
      host.innerHTML = "";

      const term = new XTerm({
        cursorBlink: true,
        cursorStyle: "block",
        fontSize: 13,
        fontFamily: '"JetBrains Mono", "Fira Code", "SFMono-Regular", Menlo, Consolas, monospace',
        scrollback: 10000,
        allowProposedApi: true,
        theme: {
          background: "#0a0a0a",
          foreground: "#d4d4d4",
          cursor: "#d4d4d4",
          cursorAccent: "#0a0a0a",
          selectionBackground: "#264f78",
          black: "#000000",
          red: "#cd3131",
          green: "#0dbc79",
          yellow: "#e5e510",
          blue: "#2472c8",
          magenta: "#bc3fbc",
          cyan: "#11a8cd",
          white: "#e5e5e5",
          brightBlack: "#666666",
          brightRed: "#f14c4c",
          brightGreen: "#23d18b",
          brightYellow: "#f5f543",
          brightBlue: "#3b8eea",
          brightMagenta: "#d670d6",
          brightCyan: "#29b8db",
          brightWhite: "#e5e5e5",
        },
      });
      term.loadAddon(new Unicode11Addon());
      const fit = new FitAddon();
      term.loadAddon(fit);
      term.loadAddon(new WebLinksAddon());
      const search = new SearchAddon();
      term.loadAddon(search);
      termRef.current = term;
      fitRef.current = fit;
      searchRef.current = search;
      term.open(host);
      term.focus();
      try {
        fit.fit();
      } catch {
        /* ainda sem dimensões */
      }

      const server = getServer();
      const wsServer = server.replace("http", "ws");
      const ws = new WebSocket(
        wsServer + "/api/v1/ws/terminal/" + appID + "?shell=" + encodeURIComponent(shellCmd)
      );
      ws.binaryType = "arraybuffer";
      wsRef.current = ws;

      const alive = () => genRef.current === gen;

      ws.onopen = () => {
        if (!alive()) return;
        setConnState("connected");
        sendResize();
      };
      ws.onmessage = (ev) => {
        if (!alive() || typeof ev.data === "string") return;
        term.write(new Uint8Array(ev.data as ArrayBuffer));
      };
      ws.onclose = () => {
        if (!alive()) return;
        term.dispose();
        if (genRef.current === gen && host) host.innerHTML = "";
        setConnState("reconnecting");
        retryRef.current = setTimeout(() => {
          if (genRef.current === gen) open(shellCmd);
        }, 1500);
      };
      ws.onerror = () => {
        if (alive()) setConnState("reconnecting");
      };

      term.onData((data) => {
        if (alive() && ws.readyState === WebSocket.OPEN) {
          ws.send(new TextEncoder().encode(data));
        }
      });

      const observer = new ResizeObserver(() => {
        try {
          fit.fit();
        } catch {
        }
        if (alive()) sendResize();
      });
      observer.observe(host);
      (host as HTMLElement & { __aetherResize?: ResizeObserver }).__aetherResize = observer;
    },
    [appID, sendResize]
  );

  useEffect(() => {
    open(SHELLS.find((s) => s.id === shell)?.cmd ?? "/bin/sh");
    return () => {
      genRef.current++;
      if (retryRef.current) clearTimeout(retryRef.current);
      wsRef.current?.close();
      termRef.current?.dispose();
      termRef.current = null;
      if (hostRef.current) hostRef.current.innerHTML = "";
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [shell, appID]);

  const statusMeta = STATUS_META[connState];

  const doSearch = (dir: "next" | "prev") => {
    const q = searchInputRef.current?.value;
    if (!q) return;
    if (dir === "next") searchRef.current?.findNext(q);
    else searchRef.current?.findPrevious(q);
  };

  return (
    <Card className="mt-lg p-0 overflow-hidden">
      <div className="flex items-center justify-between px-md py-2 border-b border-outline-variant bg-surface-container-low/50">
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1 p-0.5 bg-surface-container-high rounded-lg">
            {SHELLS.map((s) => (
              <button
                key={s.id}
                onClick={() => setShell(s.id)}
                className={`px-2.5 py-1 rounded-md font-code-md text-[12px] transition-colors ${shell === s.id ? "bg-primary text-on-primary" : "text-on-surface-variant hover:text-on-surface"}`}
              >
                {s.label}
              </button>
            ))}
          </div>
          <button
            onClick={() => setSearchOpen((v) => !v)}
            className={`flex items-center gap-1 px-2 py-1 rounded-md font-code-md text-[12px] transition-colors ${searchOpen ? "bg-primary/10 text-primary" : "text-on-surface-variant hover:text-on-surface"}`}
            title="Search terminal"
          >
            <span className="material-symbols-outlined text-[14px]">search</span>
          </button>
        </div>
        <span className="flex items-center gap-1.5 font-code-md text-[11px] text-on-surface-variant">
          <span className={`w-2 h-2 rounded-full ${statusMeta.dot}`} />
          {statusMeta.label}
        </span>
      </div>

      {searchOpen && (
        <div className="flex items-center gap-2 px-md py-2 border-b border-outline-variant bg-surface-container-low">
          <input
            ref={searchInputRef}
            autoFocus
            placeholder="Find in terminal (Enter next, Shift+Enter prev)"
            className="w-72 bg-surface-container-lowest border border-outline-variant rounded-md px-2 py-1 font-code-md text-[12px] text-on-surface placeholder:text-on-surface-variant/50 focus:border-primary focus:outline-none"
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                doSearch(e.shiftKey ? "prev" : "next");
              } else if (e.key === "Escape") {
                setSearchOpen(false);
              }
            }}
          />
          <span className="font-code-md text-[11px] text-on-surface-variant/60">Esc to close</span>
        </div>
      )}

      <div ref={hostRef} className="h-[60vh] min-h-[320px] bg-[#0a0a0a] p-1.5" />
    </Card>
  );
}
