import {
  AppHeader,
  Button,
  IconButton,
  OrganizationSwitcher,
  Sidebar,
  UserMenu,
  useTheme,
} from "@aether/design-system";
import type { Icon as DesignIcon } from "@aether/design-system";
import {
  Bell,
  Cube,
  CloudArrowUp,
  Database,
  FolderOpen,
  Gauge,
  List,
  MagnifyingGlass,
  Moon,
  Package,
  Sun,
  UsersThree,
} from "@phosphor-icons/react";
import { createElement, useEffect, useMemo, useState } from "react";
import { Outlet, useLocation, useNavigate } from "@tanstack/react-router";
import { useBranding, useMe, useProjects, useServiceDetails } from "../hooks";
import { clearToken } from "../api/client";
import { useAuthStore } from "../stores/auth";
import { CommandPalette, usePalette } from "./command-palette";
import { BellButton } from "./NotificationProvider";
import { useOrg } from "./OrgProvider";

const navGroups = [
  {
    title: "Core",
    items: [
      { label: "Projects", path: "/", icon: FolderOpen },
      { label: "Services", path: "/apps", icon: Cube },
      { label: "Monitoring", path: "/monitoring", icon: Gauge },
    ],
  },
  {
    title: "Infrastructure",
    items: [{ label: "Databases", path: "/databases", icon: Database }],
  },
  {
    title: "Storage",
    items: [{ label: "S3 Destinations", path: "/storage", icon: CloudArrowUp }],
  },
  {
    title: "Security",
    items: [{ label: "Members", path: "/members", icon: UsersThree }],
  },
  {
    title: "Platform",
    items: [
      { label: "Registry", path: "/registry", icon: Package },
      { label: "Notifications", path: "/notifications", icon: Bell },
    ],
  },
];

export function Shell() {
  const { data: me } = useMe();
  const { resolvedTheme, setTheme } = useTheme();
  const { orgs, currentOrg, switchOrg } = useOrg();
  const { data: branding } = useBranding();
  const { data: projects } = useProjects();
  const location = useLocation();
  const navigate = useNavigate();
  const appMatch = location.pathname.match(/^\/apps\/([^/]+)/);
  const projectMatch = location.pathname.match(/^\/projects\/([^/]+)/);
  const { data: serviceDetail } = useServiceDetails(appMatch?.[1] ?? "");
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(() => window.innerWidth < 768);
  const [mobileOpen, setMobileOpen] = useState(() => window.innerWidth >= 768);
  useEffect(() => {
    const media = window.matchMedia("(min-width: 768px)");
    const update = () => {
      setIsMobile(!media.matches);
      setMobileOpen(media.matches);
    };
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem("aether_sidebar_collapsed") === "1",
  );
  const activePath = location.pathname.startsWith("/apps/")
    ? "/apps"
    : location.pathname.startsWith("/projects/")
      ? "/projects"
      : location.pathname;
  const currentProject = useMemo(() => {
    const projectId = serviceDetail?.project_id ?? projectMatch?.[1];
    return projectId
      ? projects?.find((project) => project.id === projectId)?.name
      : undefined;
  }, [projectMatch, projects, serviceDetail]);
  const brandVars = branding?.primary_color
    ? ({ "--color-primary": branding.primary_color } as React.CSSProperties)
    : undefined;
  const items = navGroups.map((group) => ({
    label: group.title,
    href: group.items[0]?.path,
    children: group.items.map((item) => ({
      label: item.label,
      href: item.path,
      active: activePath === item.path,
      icon: createElement(item.icon, {
        size: 18,
        weight: activePath === item.path ? "fill" : "regular",
      }),
    })),
  }));
  const handleOrganizationChange = (orgId: string) => {
    if (orgId === currentOrg?.id) return;
    switchOrg(orgId);
    navigate({ to: "/" });
  };

  return (
    <div
      style={brandVars}
      className="flex h-dvh overflow-hidden bg-background text-foreground"
    >
      <Sidebar
        items={items}
        collapsed={collapsed}
        mobile={isMobile}
        mobileOpen={mobileOpen}
        onMobileOpenChange={setMobileOpen}
        onNavigate={(href) => {
          setMobileOpen(false);
          navigate({ to: href } as never);
        }}
        onCollapsedChange={(next) => {
          setCollapsed(next);
          localStorage.setItem("aether_sidebar_collapsed", next ? "1" : "0");
        }}
        header={
          <div className="flex items-center gap-3">
            <span className="flex size-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <CloudArrowUp size={20} weight="fill" />
            </span>
            <span>
              <strong className="block text-body-md">Aether</strong>
              <span className="block text-label-caps text-muted-foreground">
                PaaS platform
              </span>
            </span>
          </div>
        }
        footer={
          <div className="space-y-2">
            <Button
              variant="ghost"
              fullWidth
              onClick={() => setPaletteOpen(true)}
            >
              Command palette
            </Button>
            <Button
              variant="ghost"
              fullWidth
              onClick={() => {
                clearToken();
                useAuthStore.getState().clear();
                window.location.href = "/login";
              }}
            >
              Sign out
            </Button>
          </div>
        }
      />
      <div className="flex min-w-0 flex-1 flex-col">
        <AppHeader
          workspace={
            <OrganizationSwitcher
              options={orgs.map((org) => ({
                id: org.id,
                label: org.name,
                description: org.role,
              }))}
              value={currentOrg?.id}
              onValueChange={handleOrganizationChange}
              onCreate={() => navigate({ to: "/organizations/new" })}
            />
          }
          // breadcrumb={[
          //   ...(currentProject
          //     ? [{ label: currentProject, current: true }]
          //     : []),
          // ]}
          onNavigate={(href) => navigate({ to: href } as never)}
          command={
            <IconButton
              label={mobileOpen ? "Close navigation" : "Open navigation"}
              icon={List}
              className="md:hidden"
              onClick={() => setMobileOpen((open) => !open)}
            />
          }
        search={
          <IconButton
            label="Search"
            icon={MagnifyingGlass as unknown as DesignIcon}
            onClick={() => setPaletteOpen(true)}
          />
        }
        theme={
          <IconButton
            label={`Switch to ${resolvedTheme === "dark" ? "light" : "dark"} mode`}
            icon={resolvedTheme === "dark" ? Sun : Moon}
            onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
          />
        }
          notifications={<BellButton />}
          user={
            <UserMenu
              user={{
                name: me?.name ?? "User",
                email: me?.email,
                avatar: (
                  <span className="text-body-sm font-semibold text-primary">
                    {(me?.name ?? "?").slice(0, 2).toUpperCase()}
                  </span>
                ),
              }}
              currentWorkspace={currentOrg?.id}
              workspaces={orgs.map((org) => ({
                id: org.id,
                label: org.name,
                description: org.role,
              }))}
              onWorkspaceChange={switchOrg}
              onProfile={() =>
                navigate({
                  to: "/organizations/$id",
                  params: { id: currentOrg?.id ?? "" },
                } as never)
              }
              onSignOut={() => {
                clearToken();
                useAuthStore.getState().clear();
                window.location.href = "/login";
              }}
            />
          }
        />
        <main className="min-w-0 flex-1 overflow-x-hidden overflow-y-auto bg-surface-background p-4 md:p-8">
          <Outlet />
        </main>
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </div>
  );
}

export function useNavigateTo(to: string) {
  const navigate = useNavigate();
  return () => navigate({ to } as never);
}
