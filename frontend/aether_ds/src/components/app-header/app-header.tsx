import type { ReactNode } from "react";
import { Breadcrumb, type BreadcrumbItem } from "../breadcrumb/breadcrumb";
export interface AppHeaderProps {
  breadcrumb?: BreadcrumbItem[];
  workspace?: ReactNode;
  environment?: ReactNode;
  search?: ReactNode;
  command?: ReactNode;
  notifications?: ReactNode;
  theme?: ReactNode;
  user?: ReactNode;
  onNavigate?: (href: string) => void;
}
export function AppHeader({
  breadcrumb,
  command,
  environment,
  notifications,
  search,
  theme,
  user,
  workspace,
  onNavigate,
}: AppHeaderProps) {
  return (
    <header className="sticky top-0 z-30 flex min-h-16 items-center gap-4 border-b border-border bg-surface-background/75 px-4 shadow-sm backdrop-blur-xl md:px-6">
      <div className="min-w-0 flex-1 space-y-1">
        {workspace ? <div className="font-semibold">{workspace}</div> : null}
        {breadcrumb ? (
          <div className="hidden min-w-0 md:block">
            <Breadcrumb items={breadcrumb} onNavigate={onNavigate} />
          </div>
        ) : null}
      </div>
      {environment}
      {search}
      {command}
      {notifications}
      {theme}
      {user}
    </header>
  );
}
