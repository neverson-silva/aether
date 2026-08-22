import React, { useMemo } from "react";
import { Link, Outlet, useLocation, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useAppDetail, useBranding, useMe, useProjects } from "../hooks";
import { clearToken } from "../api/client";
import { useAuthStore } from "../stores/auth";
import { cn } from "./ui";
import { CommandPalette, usePalette } from "./command-palette";
import { BellButton } from "./NotificationProvider";
import { OrgSwitcher } from "./OrgSwitcher";

interface NavItem {
  label: string;
  path: string;
  icon: string;
}

interface NavGroup {
  title: string;
  items: NavItem[];
}

const NAV: NavGroup[] = [
  {
    title: "Core",
    items: [
      { label: "Projects", path: "/", icon: "folder_open" },
      { label: "Services", path: "/apps", icon: "apps" },
      { label: "Monitoring", path: "/monitoring", icon: "monitor_heart" },
      // { label: "Schedules", path: "/schedules", icon: "schedule" },
      // { label: "Requests", path: "/networking", icon: "receipt_long" },
    ],
  },
  {
    title: "Infrastructure",
    items: [
      // { label: "Clusters", path: "/clusters", icon: "hub" },
      // { label: "Nodes", path: "/servers", icon: "dns" },
      { label: "Databases", path: "/databases", icon: "storage" },
    ],
  },
  {
    title: "Storage",
    items: [
      { label: "S3 Destinations", path: "/storage", icon: "cloud_sync" },
    ],
  },
  {
    title: "Security",
    items: [
      // { label: "SSO", path: "/sso", icon: "passkey" },
      // { label: "Certificates", path: "/certificates", icon: "verified" },
      // { label: "Secrets", path: "/secrets", icon: "lock" },
      { label: "Members", path: "/members", icon: "group" },
      // { label: "API Keys", path: "/api-keys", icon: "key" },
    ],
  },
  {
    title: "Platform",
    items: [
      // { label: "Whitelabeling", path: "/whitelabel", icon: "branding_watermark" },
      { label: "Registry", path: "/registry", icon: "inventory_2" },
      { label: "Notifications", path: "/notifications", icon: "notifications_active" },
      // { label: "CI/CD", path: "/ci-cd", icon: "code_blocks" },
      // { label: "GitOps", path: "/gitops", icon: "git" },
      // { label: "Backups", path: "/backups", icon: "backup" },
      // { label: "Marketplace", path: "/marketplace", icon: "storefront" },
    ],
  },
];

export function Shell() {
  const { data: me } = useMe();
  const { data: branding } = useBranding();
  const { data: projects } = useProjects();
  const location = useLocation();
  const appMatch = location.pathname.match(/^\/apps\/([^/]+)/);
  const projectMatch = location.pathname.match(/^\/projects\/([^/]+)/);
  const { data: appDetail } = useAppDetail(appMatch?.[1] ?? "");
  const { setOpen } = usePalette();
  const [mobileNav, setMobileNav] = useState(false);
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem("aether_sidebar_collapsed") === "1");

  const toggleSidebar = () => {
    if (window.innerWidth < 1024) {
      setMobileNav(true);
      return;
    }
    setCollapsed((c) => {
      localStorage.setItem("aether_sidebar_collapsed", c ? "0" : "1");
      return !c;
    });
  };

  const activePath = useMemo(() => {
    const p = location.pathname;
    if (p.startsWith("/apps/")) return "/apps";
    if (p.startsWith("/projects/")) return "/projects";
    return p;
  }, [location.pathname]);

  const currentProject = useMemo(() => {
    if (appDetail?.app?.project_id) {
      return projects?.find((pr) => pr.id === appDetail.app.project_id)?.name;
    }
    if (projectMatch) {
      return projects?.find((pr) => pr.id === projectMatch[1])?.name;
    }
    if (location.pathname.startsWith("/projects") && projects?.length) {
      return projects[projects.length - 1].name;
    }
    return undefined;
  }, [appDetail, projects, location.pathname, projectMatch]);

  const brandVars = branding?.primary_color
    ? ({ "--color-primary": branding.primary_color } as React.CSSProperties)
    : undefined;

  return (
    <div style={brandVars} className="bg-background text-on-background h-dvh overflow-hidden flex font-body-md selection:bg-primary-container selection:text-on-primary-container">
      {mobileNav && <div className="fixed inset-0 bg-black/50 z-40 lg:hidden" onClick={() => setMobileNav(false)} />}
      <aside className={`fixed left-0 top-0 h-full w-[256px] bg-surface-container-low font-body-md text-body-md border-r border-outline-variant flex flex-col py-md px-sm z-50 transition-transform duration-200 ${mobileNav ? "translate-x-0" : "-translate-x-full"} ${collapsed ? "lg:-translate-x-full" : "lg:translate-x-0"}`}>
        <div className="flex items-center justify-between px-sm mb-md">
          <button onClick={() => setMobileNav(false)} className="lg:hidden text-on-surface-variant hover:text-on-surface">
            <span className="material-symbols-outlined">close</span>
          </button>
        </div>
        <div className="flex items-center gap-sm mb-md px-sm">
          <div className="w-8 h-8 bg-primary-container rounded flex items-center justify-center shrink-0">
            <span className="material-symbols-outlined text-on-primary-container text-[20px]">cloud</span>
          </div>
          <div>
            <h1 className="font-headline-sm text-headline-sm font-bold text-on-surface leading-none">Aether</h1>
            <p className="font-label-caps text-[10px] text-on-surface-variant mt-0.5">PaaS Platform</p>
          </div>
        </div>

        <nav className="flex-1 space-y-4 overflow-y-auto sidebar-scroll pb-4">
          <div className="px-sm">
            <OrgSwitcher />
          </div>
          {NAV.map((group) => (
            <div key={group.title} className="space-y-0.5">
              <div className="px-sm py-1 font-label-caps text-[10px] text-on-surface-variant/70 uppercase tracking-wider">
                {group.title}
              </div>
              {group.items.map((item) => {
                const active = activePath === item.path;
                return (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={cn(
                      "flex items-center gap-sm px-sm py-1.5 rounded transition-colors",
                      active
                        ? "text-primary font-bold bg-secondary-container/20"
                        : "text-on-surface-variant font-medium hover:bg-surface-container-high hover:text-on-surface"
                    )}
                  >
                    <span className="material-symbols-outlined text-[18px]">{item.icon}</span>
                    <span className="text-sm">{item.label}</span>
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>

        <div className="px-sm pt-2 border-t border-outline-variant space-y-0.5">
          <button
            onClick={() => setOpen(true)}
            className="w-full flex items-center gap-sm px-sm py-1.5 rounded text-on-surface-variant font-medium hover:bg-surface-container-high hover:text-on-surface transition-colors text-left"
          >
            <span className="material-symbols-outlined text-[18px]">search</span>
            <span className="text-sm">Commands</span>
            <kbd className="ml-auto font-code-md text-code-md text-on-surface-variant/60 border border-outline-variant rounded px-1">⌘K</kbd>
          </button>
          <button
            onClick={() => {
              clearToken();
              useAuthStore.getState().clear();
              window.location.href = "/login";
            }}
            className="w-full flex items-center gap-sm px-sm py-1.5 rounded text-on-surface-variant font-medium hover:bg-surface-container-high hover:text-on-surface transition-colors text-left"
          >
            <span className="material-symbols-outlined text-[18px]">logout</span>
            <span className="text-sm">Sign out</span>
          </button>
        </div>
      </aside>

      <header className={`fixed top-0 right-0 w-full text-primary font-label-caps text-label-caps border-b border-outline-variant flex justify-between items-center h-14 px-lg z-40 glass-panel ${collapsed ? "lg:w-full" : "lg:w-[calc(100%-256px)]"}`}>
        <div className="flex items-center gap-md min-w-0">
          <button
            onClick={toggleSidebar}
            className="text-on-surface-variant hover:text-primary transition-colors flex items-center justify-center w-8 h-8 rounded-full hover:bg-surface-container-high shrink-0"
            aria-label={collapsed ? "Open sidebar" : "Close sidebar"}
          >
            <span className="material-symbols-outlined text-[20px]">{collapsed ? "menu" : "menu_open"}</span>
          </button>
          <span className="hidden md:flex items-center gap-2 text-on-surface-variant hover:text-primary transition-colors shrink-0">
            <span className="material-symbols-outlined text-[16px]">cloud</span>
            Aether
          </span>
          <span className="hidden md:block material-symbols-outlined text-[16px] text-on-surface-variant/60 shrink-0">chevron_right</span>
          <span className="text-on-surface border-b-2 border-primary pb-1 hover:text-primary transition-colors truncate">
            {me?.org?.name || "org"}
          </span>
          {currentProject && (
            <>
              <span className="hidden md:block material-symbols-outlined text-[16px] text-on-surface-variant/60 shrink-0">chevron_right</span>
              <span className="hidden md:inline text-on-surface-variant hover:text-primary transition-colors truncate max-w-[160px]">
                {currentProject}
              </span>
            </>
          )}
        </div>
        <div className="flex items-center gap-md">
          <button
            onClick={() => setOpen(true)}
            className="text-on-surface-variant hover:text-primary transition-colors flex items-center justify-center w-8 h-8 rounded-full hover:bg-surface-container-high"
            title="Search (⌘K)"
          >
            <span className="material-symbols-outlined text-[18px]">search</span>
          </button>
          <BellButton />
          <button className="text-on-surface-variant hover:text-primary transition-colors flex items-center justify-center w-8 h-8 rounded-full hover:bg-surface-container-high">
            <span className="material-symbols-outlined text-[18px]">help_outline</span>
          </button>
          <div className="w-8 h-8 rounded-full bg-primary-container flex items-center justify-center">
            <span className="font-label-caps text-label-caps text-on-primary-container">
              {(me?.name || "?").slice(0, 2).toUpperCase()}
            </span>
          </div>
        </div>
      </header>

      <main className={`flex-1 flex flex-col h-full relative min-w-0 ${collapsed ? "lg:ml-0" : "lg:ml-[256px]"}`}>
        <div className="flex-1 overflow-y-auto mt-14 p-margin-desktop bg-surface-dim">
          <div className="max-w-[1600px] mx-auto space-y-lg">
            <Outlet />
          </div>
        </div>
      </main>

      <CommandPalette />
    </div>
  );
}

export function useNavigateTo(to: string) {
  const navigate = useNavigate();
  return () => navigate({ to } as never);
}
