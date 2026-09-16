import { Code, Link, PencilSimple, Trash } from "@phosphor-icons/react";
import { AlertDialog, Button, RuntimeStatus, type RuntimeStatusValue } from "@aether/design-system";
import type { App, ServiceSummary } from "@/api/types";
import type { ServiceTab } from "./service-tabs";
import { SERVICE_TAB_ICONS } from "./service-tabs";

export function ServiceHeader({
  app,
  service,
  runtimeId,
  runtimeStatus,
  isLive,
  visibleTabs,
  tab,
  onTabChange,
  onEdit,
  onDelete,
  deletePending = false,
}: {
  app: App;
  service: ServiceSummary;
  runtimeId: string;
  runtimeStatus: RuntimeStatusValue;
  isLive: boolean;
  visibleTabs: readonly ServiceTab[];
  tab: ServiceTab;
  onTabChange: (tab: ServiceTab) => void;
  onEdit: () => void;
  onDelete: () => void;
  deletePending?: boolean;
}) {
  return (
    <>
      <div className="mb-8 flex flex-col justify-between gap-6 md:flex-row md:items-end">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            {app.source_type === "git" ? <Code size={32} className="shrink-0 text-primary" /> : <Link size={32} className="shrink-0 text-primary" />}
            <h2 className="truncate font-display-lg text-[clamp(1.5rem,4vw,3rem)] leading-[1.1] text-on-surface">{app.name}</h2>
            <RuntimeStatus status={runtimeStatus} live={isLive} />
            <p className="font-body-md text-body-md text-on-surface-variant md:ml-4">:{app.port}</p>
          </div>
          <p className="mt-1 max-w-[42rem] font-body-md text-body-md text-on-surface-variant">{app.source_type === "image" ? app.image : app.git_url}</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {service.kind === "database" ? <Button variant="primary" onClick={() => { window.location.href = `/studio/${runtimeId}`; }}>Open Studio</Button> : null}
          <div className="flex gap-2">
            <button type="button" className="text-on-surface-variant transition-colors hover:text-primary" onClick={onEdit} title="Edit service name" aria-label="Edit service name">
              <PencilSimple size={18} />
            </button>
            <AlertDialog
              trigger={<button type="button" className="text-on-surface-variant transition-colors hover:text-error" title="Delete service" aria-label="Delete service"><Trash size={18} /></button>}
              onConfirm={onDelete}
              title="Delete service"
              description={`Remove ${service.name} and all deployments? Active containers will be stopped.`}
              confirmLabel="Delete"
              confirmDisabled={deletePending}
            />
          </div>
        </div>
      </div>
      <div className="mb-8 flex max-w-full snap-x gap-6 overflow-x-auto overscroll-x-contain border-b border-outline-variant pb-px" role="tablist" aria-label="Service sections">
        {visibleTabs.map((item) => {
          const Icon = SERVICE_TAB_ICONS[item];
          return (
            <button
              key={item}
              type="button"
              role="tab"
              aria-selected={tab === item}
              onClick={() => onTabChange(item)}
              className={`inline-flex min-w-max items-center gap-2 whitespace-nowrap px-1 pb-3 font-label-caps text-label-caps capitalize transition-colors ${tab === item ? "border-b-2 border-primary text-primary" : "text-on-surface-variant hover:text-on-surface"}`}
            >
              <Icon size={16} aria-hidden="true" />
              {item}
            </button>
          );
        })}
      </div>
    </>
  );
}
