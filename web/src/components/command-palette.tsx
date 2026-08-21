import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useProjects } from "../hooks";
import { getServer } from "../api/client";
import { CreateServiceLauncher } from "./CreateServiceLauncher";
import { useOverlayGate } from "./OverlayManager";
import { usePaletteStore } from "../stores/palette";

export function usePalette() {
  const open = usePaletteStore((s) => s.open);
  const setOpen = usePaletteStore((s) => s.setOpen);
  return { open, setOpen };
}

interface NavEntry {
  label: string;
  path: string;
  icon: string;
  hint: string;
}

const NAV: NavEntry[] = [
  { label: "Projects", path: "/", icon: "folder_open", hint: "Go to" },
  { label: "Services", path: "/apps", icon: "apps", hint: "Go to" },
  { label: "Databases", path: "/databases", icon: "storage", hint: "Go to" },
  { label: "Marketplace", path: "/marketplace", icon: "storefront", hint: "Go to" },
  { label: "Monitoring", path: "/monitoring", icon: "monitor_heart", hint: "Go to" },
  { label: "Schedules", path: "/schedules", icon: "schedule", hint: "Go to" },
  { label: "Requests", path: "/networking", icon: "receipt_long", hint: "Go to" },
  { label: "Clusters", path: "/clusters", icon: "hub", hint: "Go to" },
  { label: "Nodes", path: "/servers", icon: "dns", hint: "Go to" },
  { label: "Certificates", path: "/certificates", icon: "verified", hint: "Go to" },
  { label: "Secrets", path: "/secrets", icon: "lock", hint: "Go to" },
  { label: "Members", path: "/members", icon: "group", hint: "Go to" },
  { label: "API Keys", path: "/api-keys", icon: "key", hint: "Go to" },
  { label: "SSO", path: "/sso", icon: "passkey", hint: "Go to" },
  { label: "Whitelabeling", path: "/whitelabel", icon: "branding_watermark", hint: "Go to" },
  { label: "Registry", path: "/registry", icon: "inventory_2", hint: "Go to" },
  { label: "Notifications", path: "/notifications", icon: "notifications_active", hint: "Go to" },
  { label: "CI/CD", path: "/ci-cd", icon: "code_blocks", hint: "Go to" },
  { label: "GitOps", path: "/gitops", icon: "git", hint: "Go to" },
  { label: "Backups", path: "/backups", icon: "backup", hint: "Go to" },
  { label: "S3 Destinations", path: "/storage", icon: "cloud_sync", hint: "Go to" },
];

const CREATE: { label: string; icon: string; action: "service" | "database" | "environment" | "project" | "cron" | "worker" }[] = [
  { label: "Service", icon: "rocket_launch", action: "service" },
  { label: "Database", icon: "storage", action: "database" },
  { label: "Environment", icon: "deployed_code", action: "environment" },
  { label: "Project", icon: "folder_open", action: "project" },
  { label: "Cron Job", icon: "schedule", action: "cron" },
  { label: "Worker", icon: "bolt", action: "worker" },
];

const RECENTS_KEY = "aether.launcher.recents";
const FAVS_KEY = "aether.launcher.favs";

function loadJSON(key: string): string[] {
  try {
    return JSON.parse(localStorage.getItem(key) || "[]");
  } catch {
    return [];
  }
}

function saveJSON(key: string, v: string[]) {
  localStorage.setItem(key, JSON.stringify(v.slice(0, 5)));
}

function pushRecent(key: string, label: string) {
  const recents = loadJSON(RECENTS_KEY).filter((r) => r !== label);
  recents.unshift(label);
  saveJSON(RECENTS_KEY, recents);
  void key;
}

interface PaletteItem {
  id: string;
  section: string;
  label: string;
  sublabel?: string;
  icon: string;
  kind: "nav" | "create" | "template";
  payload?: string;
  action?: "service" | "database" | "environment" | "project" | "cron" | "worker";
}

export function CommandPaletteProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

export function CommandPalette() {
  const { open, setOpen } = usePalette();
  const { mounted, closing, close } = useOverlayGate("command-palette", open, () => setOpen(false));
  const [query, setQuery] = useState("");
  const [index, setIndex] = useState(0);
  const [serviceCreate, setServiceCreate] = useState(false);
  const [templates, setTemplates] = useState<PaletteItem[]>([]);
  const navigate = useNavigate();
  const { data: projects } = useProjects();

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen(!open);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, setOpen]);

  useEffect(() => {
    if (!open) {
      setQuery("");
      setIndex(0);
    }
  }, [open]);

  const mode = query.startsWith(">") ? "commands" : query.startsWith("@") ? "projects" : query.startsWith("#") ? "templates" : "all";
  const modeQuery = mode === "all" ? query.trim().toLowerCase() : query.slice(1).trim().toLowerCase();

  useEffect(() => {
    if (mode === "templates" && templates.length === 0) {
      fetch(`${getServer()}/api/v1/templates`, {
        credentials: "include",
      })
        .then((r) => r.json())
        .then((list: { id: string; name: string; description: string; icon: string; category: string }[]) => {
          setTemplates(
            list.map((t) => ({
              id: t.id,
              section: "Templates",
              label: t.name,
              sublabel: t.description,
              icon: t.icon || "dashboard",
              kind: "template",
              payload: t.id,
            }))
          );
        })
        .catch(() => setTemplates([]));
    }
  }, [mode, templates.length]);

  const items = useMemo<PaletteItem[]>(() => {
    if (mode === "templates") {
      return templates.filter(
        (t) => !modeQuery || t.label.toLowerCase().includes(modeQuery) || (t.sublabel ?? "").toLowerCase().includes(modeQuery)
      );
    }
    if (mode === "projects") {
      return (projects ?? [])
        .filter((p) => !modeQuery || p.name.toLowerCase().includes(modeQuery))
        .map((p) => ({
          id: "proj-" + p.id,
          section: "Projects",
          label: p.name,
          icon: "folder_open",
          kind: "nav" as const,
          payload: `/projects/${p.id}`,
        }));
    }
    const favs = new Set(loadJSON(FAVS_KEY));
    const navItems: PaletteItem[] = [];
    const recents = loadJSON(RECENTS_KEY);
    if (!modeQuery) {
      for (const r of recents) {
        const match = NAV.find((n) => n.label === r);
        if (match) {
          navItems.push({ id: "rec-" + match.path, section: "Recents", label: match.label, icon: match.icon, kind: "nav", payload: match.path });
        }
      }
    }
    if (mode === "commands") {
      for (const c of NAV) {
        if (favs.has(c.label)) {
          navItems.push({ id: "nav-" + c.path, section: "Favorite", label: c.label, icon: c.icon, kind: "nav", payload: c.path, sublabel: "Command" });
        }
      }
    }
    for (const c of CREATE) {
      if (!modeQuery || c.label.toLowerCase().includes(modeQuery)) {
        navItems.push({ id: "create-" + c.action, section: "Create", label: "Create " + c.label, icon: c.icon, kind: "create", action: c.action });
      }
    }
    for (const n of NAV) {
      if (!modeQuery || n.label.toLowerCase().includes(modeQuery)) {
        navItems.push({ id: "nav-" + n.path, section: "Navigate", label: n.label, icon: n.icon, kind: "nav", payload: n.path, sublabel: n.hint });
      }
    }
    return navItems;
  }, [mode, modeQuery, projects, templates]);

  useEffect(() => {
    setIndex(0);
  }, [query]);

  const run = (item: PaletteItem) => {
    close();
    if (item.kind === "nav" && item.payload) {
      pushRecent("", item.label);
      navigate({ to: item.payload } as never);
    } else if (item.kind === "create") {
      if (item.action === "service") {
        setServiceCreate(true);
      } else if (item.action === "project") {
        navigate({ to: "/projects/new" } as never);
      } else if (item.action === "database") {
        navigate({ to: "/databases" } as never);
      } else if (item.action === "environment") {
        navigate({ to: "/projects" } as never);
      } else if (item.action === "cron" || item.action === "worker") {
        navigate({ to: "/apps" } as never);
      }
    } else if (item.kind === "template" && item.payload) {
      navigate({ to: "/marketplace", search: { q: item.label } } as never);
    }
  };

  const toggleFav = (item: PaletteItem) => {
    const favs = new Set(loadJSON(FAVS_KEY));
    if (favs.has(item.label)) favs.delete(item.label);
    else favs.add(item.label);
    localStorage.setItem(FAVS_KEY, JSON.stringify([...favs]));
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setIndex((i) => Math.min(i + 1, items.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter" && items[index]) {
      run(items[index]);
    }
  };

  const placeholder =
    mode === "commands" ? "Type a command..." : mode === "projects" ? "Filter by project..." : mode === "templates" ? "Search templates..." : "Search or create...";

  return (
    <>
    {mounted && (
    <div className={`fixed inset-0 z-[90] flex items-start justify-center pt-[18vh] bg-black/60 backdrop-blur-[2px] ${closing ? "animate-fade-out" : "animate-fade-in"}`} onClick={() => close()}>
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Command launcher"
        aria-live="polite"
        onClick={(e) => e.stopPropagation()}
        className={`w-full max-w-[620px] bg-surface-modal border border-border-subtle rounded-2xl shadow-xl overflow-hidden ${closing ? "animate-fade-out-up" : "animate-modal-pop"}`}
      >
        <div className="flex items-center gap-sm px-md py-3.5 border-b border-border-subtle">
          <span className="material-symbols-outlined text-[18px] text-on-surface-variant">{mode === "commands" ? "bolt" : mode === "projects" ? "folder_open" : mode === "templates" ? "storefront" : "search"}</span>
          <input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder={placeholder}
            className="flex-1 bg-transparent font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant/50 focus:outline-none"
          />
          <kbd className="font-code-md text-code-md text-on-surface-variant/60 border border-outline-variant rounded px-1.5">esc</kbd>
        </div>
        {mode === "all" && !modeQuery && (
          <div className="px-md pt-2 font-code-md text-code-md text-on-surface-variant/50">
            Type <span className="text-primary">&gt;</span> commands · <span className="text-primary">@</span> projects · <span className="text-primary">#</span> templates
          </div>
        )}
        <div className="max-h-[380px] overflow-y-auto py-1.5">
          {items.length === 0 && (
            <p className="px-md py-3 font-body-sm text-body-sm text-on-surface-variant">No results for "{query}"</p>
          )}
          {items.map((item, i) => (
            <div key={item.id} className={`px-2 ${i === 0 || item.section !== items[i - 1]?.section ? "mt-1" : ""}`}>
              {i === 0 || item.section !== items[i - 1]?.section ? (
                <p className="px-2.5 pt-2 pb-1 font-label-caps text-label-caps text-on-surface-variant/60 uppercase">{item.section}</p>
              ) : null}
              <button
                onMouseEnter={() => setIndex(i)}
                onClick={() => run(item)}
                className={`w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-colors ${
                  i === index ? "bg-surface-container-high" : "hover:bg-surface-container-high/60"
                }`}
              >
                <span className={`material-symbols-outlined text-[16px] shrink-0 ${item.kind === "create" ? "text-primary" : "text-on-surface-variant"}`}>{item.icon}</span>
                <span className="flex-1 min-w-0">
                  <span className="block font-body-md text-body-md text-on-surface truncate">{item.label}</span>
                  {item.sublabel && <span className="block font-code-md text-code-md text-on-surface-variant/60 truncate">{item.sublabel}</span>}
                </span>
                {item.kind === "nav" && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleFav(item);
                    }}
                    className={`material-symbols-outlined text-[16px] shrink-0 ${loadJSON(FAVS_KEY).includes(item.label) ? "text-[#fbbf24]" : "text-on-surface-variant/40"}`}
                    aria-label="Toggle favorite"
                  >
                    star
                  </button>
                )}
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
    )}
    <CreateServiceLauncher open={serviceCreate} onClose={() => setServiceCreate(false)} />
    </>
  );
}

