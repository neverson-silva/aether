import {
  CommandPalette,
  Popover,
  ToastProvider,
  Tooltip,
  UserMenu,
  WorkspaceSwitcher,
} from '@aether/elisyum-ds'
import {
  Archive,
  ArrowsInLineHorizontal,
  ArrowsOutLineHorizontal,
  Bell,
  ChartLineUp,
  CloudArrowUp,
  Command,
  Database,
  DotsThree,
  FileCode,
  GlobeHemisphereWest,
  House,
  MagnifyingGlass,
  Package,
  Stack,
  UsersThree,
} from '@phosphor-icons/react'
import { useQueryClient } from '@tanstack/react-query'
import { Link, Outlet, useLocation, useNavigate } from '@tanstack/react-router'
import { type ReactNode, useEffect, useState } from 'react'
import { logout } from './api/client'
import {
  NotificationBell,
  NotificationProvider,
} from './components/NotificationProvider'
import { useOrg } from './components/OrgProvider'
import { PageContainer, type PageWidth } from './components/page-container'
import { useMe } from './hooks/use-me'
import { useAuthStore } from './stores/auth'

type RailItem = { label: string; href: string; icon: ReactNode }

const railItems: RailItem[] = [
  { label: 'Home', href: '/', icon: <House size={19} /> },
  { label: 'Projects', href: '/projects', icon: <Stack size={19} /> },
  { label: 'Services', href: '/apps', icon: <Package size={19} /> },
  { label: 'Monitoring', href: '/monitoring', icon: <ChartLineUp size={19} /> },
  { label: 'Databases', href: '/databases', icon: <Database size={19} /> },
  { label: 'Networking', href: '/networking', icon: <GlobeHemisphereWest size={19} /> },
  { label: 'S3 Destinations', href: '/storage', icon: <CloudArrowUp size={19} /> },
  { label: 'Members', href: '/members', icon: <UsersThree size={19} /> },
  { label: 'Registry', href: '/registry', icon: <Archive size={19} /> },
  { label: 'Traefik File System', href: '/traefik', icon: <FileCode size={19} /> },
  { label: 'Notifications', href: '/notifications', icon: <Bell size={19} /> },
]

function isActive(pathname: string, href: string) {
  return href === '/' ? pathname === '/' : pathname.startsWith(href)
}

function contentWidth(pathname: string, search: string): PageWidth {
  if (pathname === '/') return 'wide'
  if (pathname.startsWith('/projects/') && pathname !== '/projects/new') return 'full'
  if (
    pathname === '/monitoring' ||
    pathname === '/apps' ||
    pathname === '/databases' ||
    pathname === '/registry'
  )
    return 'wide'
  if (pathname === '/traefik') return 'full'
  if (pathname.startsWith('/services/')) {
    const tab = new URLSearchParams(search).get('tab')
    if (tab === 'deployments' || tab === 'logs' || tab === 'metrics') return 'wide'
    return 'standard'
  }
  if (pathname === '/services/new' || pathname === '/projects/new') return 'readable'
  if (pathname.startsWith('/databases/')) return 'standard'
  return 'standard'
}

function CommandRail({
  expanded,
  onToggle,
}: {
  expanded: boolean
  onToggle: () => void
}) {
  const { pathname } = useLocation()
  return (
    <aside
      className={`fixed inset-x-auto inset-y-0 left-0 z-40 hidden min-w-0 overflow-x-hidden flex-col border-r border-border-subtle bg-surface-0 transition-[width] duration-[var(--ely-duration-overlay)] lg:flex ${expanded ? 'w-64' : 'w-[4.5rem]'}`}
    >
      <div
        className={`flex h-16 items-center border-b border-border-subtle ${expanded ? 'justify-start px-4' : 'justify-center px-2'}`}
      >
        <Link
          aria-label="Aether home"
          className="flex min-w-0 items-center gap-2 text-text-primary"
          to="/"
        >
          <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-action text-on-action">
            <Command
              size={18}
              weight="bold"
            />
          </span>
          {expanded ? (
            <span className="truncate text-label tracking-[0.14em]">AETHER</span>
          ) : null}
        </Link>
      </div>
      <nav
        aria-label="Primary navigation"
        className={`sidebar-scroll-area grid min-h-0 min-w-0 flex-1 content-start gap-2 overflow-x-hidden overflow-y-auto py-5 ${expanded ? 'px-3' : 'justify-items-center'}`}
      >
        {railItems.map((item) => (
          <RailLink
            expanded={expanded}
            item={item}
            active={isActive(pathname, item.href)}
            key={item.href}
          />
        ))}
      </nav>
      <div className={`mt-auto p-3 ${expanded ? '' : 'grid justify-items-center'}`}>
        <button
          aria-expanded={expanded}
          aria-label={expanded ? 'Collapse sidebar' : 'Expand sidebar'}
          className={`grid min-h-10 place-items-center rounded-lg text-text-tertiary transition-colors hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus ${expanded ? 'w-full' : 'size-10'}`}
          onClick={onToggle}
          type="button"
        >
          {expanded ? (
            <ArrowsInLineHorizontal size={18} />
          ) : (
            <ArrowsOutLineHorizontal size={18} />
          )}
        </button>
      </div>
    </aside>
  )
}

function RailLink({
  expanded,
  item,
  active,
}: {
  expanded: boolean
  item: RailItem
  active: boolean
}) {
  const link = (
    <Link
      aria-current={active ? 'page' : undefined}
      aria-label={item.label}
      className={`group relative flex min-h-11 min-w-0 items-center rounded-lg text-text-secondary transition-[background-color,border-color,color] duration-[var(--ely-duration-fast)] ${expanded ? 'w-full gap-3 px-3' : 'size-11 justify-center'} ${active ? 'border border-action/40 bg-surface-2 text-action-strong before:absolute before:inset-y-2 before:left-0 before:w-0.5 before:rounded-full before:bg-action' : 'border border-transparent hover:bg-surface-2 hover:text-text-primary'}`}
      to={item.href}
    >
      {item.icon}
      {expanded ? <span className="truncate text-supporting">{item.label}</span> : null}
    </Link>
  )
  return expanded ? link : <Tooltip trigger={link}>{item.label}</Tooltip>
}

function ScopeBand() {
  const { currentOrg, orgs, switchOrg } = useOrg()
  const navigate = useNavigate()
  const { data: me } = useMe()
  const queryClient = useQueryClient()
  const userName = me?.name || me?.email || 'Operator'
  const handleLogout = async () => {
    await logout()
    useAuthStore.getState().clear()
    await queryClient.clear()
    window.location.assign('/login')
  }
  return (
    <div className="sticky top-0 z-30 flex h-16 min-w-0 items-center justify-between gap-2 border-b border-border-subtle/70 bg-surface-1/90 px-3 backdrop-blur-xl sm:gap-4 sm:px-6 lg:mt-3 lg:h-[4.5rem] lg:rounded-2xl lg:border lg:px-8">
      <div className="min-w-0 flex-1 overflow-hidden">
        <WorkspaceSwitcher
          options={orgs.map((org) => ({
            id: org.id,
            name: org.name || org.slug || org.id,
            detail: org.role,
          }))}
          value={currentOrg?.id}
          onValueChange={switchOrg}
        />
      </div>
      <div className="flex shrink-0 items-center gap-0.5 sm:gap-2">
        <CommandPalette
          title="Search resources"
          triggerClassName="inline-flex size-9 cursor-pointer items-center justify-center rounded-lg bg-transparent p-0 text-text-secondary transition-[background-color,color,transform] duration-[var(--ely-duration-fast)] ease-ely-out hover:bg-surface-2 hover:text-text-primary active:bg-surface-3 motion-safe:active:scale-[var(--ely-motion-press-scale)] focus-visible:outline-2 focus-visible:outline-focus sm:size-10 sm:rounded-xl"
          trigger={
            <MagnifyingGlass
              aria-label="Search resources"
              size={18}
            />
          }
          items={[
            {
              id: 'projects',
              label: 'Open projects',
              keywords: ['project', 'inventory'],
              onSelect: () => {
                void navigate({ to: '/projects' })
              },
            },
            {
              id: 'services',
              label: 'Open services',
              keywords: ['service', 'runtime'],
              onSelect: () => {
                void navigate({ to: '/apps' })
              },
            },
            {
              id: 'databases',
              label: 'Open databases',
              keywords: ['database', 'storage'],
              onSelect: () => {
                void navigate({ to: '/databases' })
              },
            },
            {
              id: 'monitoring',
              label: 'Open monitoring',
              keywords: ['observability', 'telemetry'],
              onSelect: () => {
                void navigate({ to: '/monitoring' })
              },
            },
          ]}
        />
        <NotificationBell />
        <span className="mx-1 h-6 w-px bg-border-subtle/70 sm:mx-2" />
        <UserMenu
          compact
          email={me?.email}
          name={userName}
          items={[
            {
              label: 'Sign out',
              danger: true,
              onSelect: () => {
                void handleLogout()
              },
            },
          ]}
        />
      </div>
    </div>
  )
}

function MobileBand() {
  const { pathname } = useLocation()
  const [moreOpen, setMoreOpen] = useState(false)
  const primaryItems = railItems.slice(0, 4)
  const secondaryItems = railItems.slice(4)
  const secondaryActive = secondaryItems.some((item) => isActive(pathname, item.href))
  return (
    <nav
      aria-label="Mobile navigation"
      className="fixed inset-x-0 bottom-0 z-40 grid grid-cols-5 border-t border-border-subtle/70 bg-surface-1/90 px-2 pb-2 pt-1.5 shadow-[0_-12px_32px_rgb(0_0_0_/_0.22)] backdrop-blur-xl lg:hidden"
    >
      {primaryItems.map((item) => (
        <Link
          aria-current={isActive(pathname, item.href) ? 'page' : undefined}
          className={`grid min-h-12 place-items-center gap-0.5 rounded-xl border text-[0.68rem] ${isActive(pathname, item.href) ? 'border-action/40 bg-action-soft text-action-strong' : 'border-transparent text-text-secondary hover:bg-surface-2'}`}
          key={item.href}
          to={item.href}
        >
          <span>{item.icon}</span>
          <span>{item.label}</span>
        </Link>
      ))}
      <Popover
        align="end"
        className="!w-[calc(100vw-1.5rem)] !min-w-0 !rounded-2xl !border-border-subtle !bg-surface-1 !p-2 !shadow-elevation-4"
        onOpenChange={setMoreOpen}
        open={moreOpen}
        side="top"
        sideOffset={8}
        trigger={
          <span
            className={`grid min-h-12 place-items-center gap-0.5 rounded-xl border text-[0.68rem] ${moreOpen || secondaryActive ? 'border-action/40 bg-action-soft text-action-strong' : 'border-transparent text-text-secondary hover:bg-surface-2'}`}
          >
            <DotsThree
              aria-hidden="true"
              size={20}
              weight="bold"
            />
            <span>More</span>
          </span>
        }
        triggerClassName="w-full"
      >
        <div className="grid gap-1">
          {secondaryItems.map((item) => (
            <Link
              aria-current={isActive(pathname, item.href) ? 'page' : undefined}
              className={`flex min-h-11 items-center gap-3 rounded-xl px-3 text-supporting transition-colors ${isActive(pathname, item.href) ? 'bg-action-soft text-action-strong' : 'text-text-secondary hover:bg-surface-2 hover:text-text-primary'}`}
              key={item.href}
              onClick={() => setMoreOpen(false)}
              to={item.href}
            >
              <span className="grid size-8 place-items-center rounded-lg bg-surface-2">
                {item.icon}
              </span>
              <span>{item.label}</span>
            </Link>
          ))}
        </div>
      </Popover>
    </nav>
  )
}

export function AppShell() {
  const [sidebarExpanded, setSidebarExpanded] = useState(
    () =>
      typeof window === 'undefined' ||
      window.localStorage.getItem('aether_sidebar_expanded') !== 'false',
  )
  const { pathname, searchStr } = useLocation()
  useEffect(() => {
    window.localStorage.setItem('aether_sidebar_expanded', String(sidebarExpanded))
  }, [sidebarExpanded])
  return (
    <ToastProvider>
      <NotificationProvider>
        <div className="min-h-screen bg-canvas text-text-primary">
          <CommandRail
            expanded={sidebarExpanded}
            onToggle={() => setSidebarExpanded((value) => !value)}
          />
          <MobileBand />
          <div
            className={`min-w-0 transition-[padding] duration-[var(--ely-duration-overlay)] ${sidebarExpanded ? 'lg:pl-64' : 'lg:pl-[4.5rem]'}`}
          >
            <div className="mx-0 w-full max-w-none sm:px-6 lg:px-8">
              <ScopeBand />
              <main className="mx-0 w-full max-w-none px-4 pb-24 pt-5 sm:px-0 lg:py-7 lg:pb-0">
                <PageContainer width={contentWidth(pathname, searchStr)}>
                  <Outlet />
                </PageContainer>
              </main>
            </div>
          </div>
        </div>
      </NotificationProvider>
    </ToastProvider>
  )
}
