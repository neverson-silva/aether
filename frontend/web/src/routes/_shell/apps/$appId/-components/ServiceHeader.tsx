import { Code, Link, PencilSimple, Trash } from "@phosphor-icons/react";
import type { KeyboardEvent } from "react";
import {
  AlertDialog,
  Button,
  RuntimeStatus,
  type RuntimeStatusValue,
} from "@aether/design-system";
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
  const handleTabKeyDown = (
    event: KeyboardEvent<HTMLButtonElement>,
    index: number,
  ) => {
    if (
      event.key !== "ArrowRight" &&
      event.key !== "ArrowLeft" &&
      event.key !== "Home" &&
      event.key !== "End"
    )
      return;
    event.preventDefault();
    const nextIndex =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? visibleTabs.length - 1
          : (index +
              (event.key === "ArrowRight" ? 1 : -1) +
              visibleTabs.length) %
            visibleTabs.length;
    const nextTab = visibleTabs[nextIndex];
    onTabChange(nextTab);
    document.getElementById(`service-tab-${nextTab}`)?.focus();
  };

  return (
    <>
      <section className="relative isolate overflow-hidden rounded-2xl border border-outline-variant bg-gradient-to-br from-primary/10 via-surface-container-low to-secondary/5 p-lg shadow-md">
        <div
          className="pointer-events-none absolute -right-24 -top-32 -z-10 size-80 rounded-full bg-primary/10 blur-3xl"
          aria-hidden="true"
        />
        <div
          className="pointer-events-none absolute -bottom-40 left-1/3 -z-10 size-72 rounded-full bg-secondary/10 blur-3xl"
          aria-hidden="true"
        />
        <div className="relative flex flex-col justify-between gap-lg md:flex-row md:items-end">
          <div className="min-w-0">
            <div className="mb-sm flex flex-wrap items-center gap-sm font-label-caps text-label-caps text-on-surface-variant">
              <span className="inline-flex items-center gap-xs rounded-full border border-outline-variant bg-surface-container/70 px-sm py-1 text-primary">
                {app.source_type === "git" ? (
                  <Code size={14} aria-hidden="true" />
                ) : (
                  <Link size={14} aria-hidden="true" />
                )}
                Service
              </span>
              <span className="text-on-surface-variant/60">/</span>
              <RuntimeStatus status={runtimeStatus} live={isLive} />
            </div>
            <div className="flex flex-wrap items-end gap-sm">
              <h2 className="max-w-full truncate font-display-lg text-[clamp(1.75rem,5vw,3.5rem)] leading-[1.02] tracking-[-0.04em] text-on-surface">
                {app.name}
              </h2>
              <span className="mb-1 inline-flex items-center rounded-full border border-outline-variant bg-surface-container/70 px-sm py-1 font-code-md text-code-md text-on-surface-variant">
                :{app.port}
              </span>
            </div>
            <p
              className="mt-md max-w-[48rem] truncate rounded-lg border border-outline-variant/70 bg-surface-container-lowest/50 px-sm py-1.5 font-code-md text-code-md text-on-surface-variant"
              title={app.source_type === "image" ? app.image : app.git_url}
            >
              {app.source_type === "image" ? app.image : app.git_url}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-sm md:justify-end">
            {service.kind === "database" ? (
              <Button
                variant="primary"
                disabled={!isLive}
                title={
                  isLive
                    ? "Open database studio"
                    : "Studio becomes available when the database is ready"
                }
                onClick={() => {
                  window.location.href = `/studio/${runtimeId}`;
                }}
              >
                Open Studio
              </Button>
            ) : null}
            <div className="flex items-center gap-xs rounded-xl border border-outline-variant bg-surface-container/80 p-1">
              <button
                type="button"
                className="rounded-md p-2 text-on-surface-variant outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container-high hover:text-primary active:scale-[0.96] focus-visible:ring-2 focus-visible:ring-ring"
                onClick={onEdit}
                title="Edit service name"
                aria-label="Edit service name"
              >
                <PencilSimple size={18} aria-hidden="true" />
              </button>
              <AlertDialog
                trigger={
                  <button
                    type="button"
                    className="rounded-md p-2 text-on-surface-variant outline-none transition-[background-color,color,transform] duration-150 hover:bg-error/10 hover:text-error active:scale-[0.96] focus-visible:ring-2 focus-visible:ring-ring"
                    title="Delete service"
                    aria-label="Delete service"
                  >
                    <Trash size={18} aria-hidden="true" />
                  </button>
                }
                onConfirm={onDelete}
                title="Delete service"
                description={`Remove ${service.name} and all deployments? Active containers will be stopped.`}
                confirmLabel="Delete"
                confirmDisabled={deletePending}
              />
            </div>
          </div>
        </div>
        <div
          className="relative mt-lg flex max-w-full snap-x gap-1 overflow-x-auto overscroll-x-contain rounded-xl border border-outline-variant bg-surface-container-lowest/70 p-1 backdrop-blur-sm"
          role="tablist"
          aria-label="Service sections"
        >
          {visibleTabs.map((item) => {
            const Icon = SERVICE_TAB_ICONS[item];
            return (
              <button
                key={item}
                type="button"
                role="tab"
                id={`service-tab-${item}`}
                aria-selected={tab === item}
                aria-controls={`service-panel-${item}`}
                tabIndex={tab === item ? 0 : -1}
                onClick={() => onTabChange(item)}
                onKeyDown={(event) =>
                  handleTabKeyDown(event, visibleTabs.indexOf(item))
                }
                className={`inline-flex min-w-max items-center gap-2 whitespace-nowrap rounded-md px-3 py-2 font-label-caps text-label-caps capitalize outline-none transition-[background-color,border-color,color,box-shadow,transform] duration-150 active:scale-[0.98] focus-visible:ring-2 focus-visible:ring-ring ${tab === item ? "bg-surface-container-high text-primary shadow-sm" : "text-on-surface-variant hover:bg-surface-container hover:text-on-surface"}`}
              >
                <Icon size={16} aria-hidden="true" />
                {item}
              </button>
            );
          })}
        </div>
      </section>
    </>
  );
}
