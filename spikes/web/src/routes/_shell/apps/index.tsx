import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useAppStates, useDatabases } from "../../../api/hooks";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link } from "@tanstack/react-router";
import { useApps, useCreateApp, useProjects } from "../../../api/hooks";
import { CreateServiceLauncher } from "../../../components/CreateServiceLauncher";
import { TechIcon } from "../../../components/TechIcon";
import { AppPage, AppPageHeader } from "../../../components/ds";
import {
  Button,
  Card,
  EmptyState, SkeletonList,
  Field,
  Input,
  Modal,
  Select,
  StatusPill,
  useToast,
} from "../../../components/ui";

function Apps() {
  const { data: apps, isLoading } = useApps();
  const { data: states } = useAppStates();
  const { data: databases } = useDatabases();
  const { data: projects } = useProjects();
  const [open, setOpen] = useState(false);
  const { toast } = useToast();
  return (
    <AppPage>
      <AppPageHeader
        title="Services"
        description="Deployed services: OCI images or git repositories. Click a service to open its detail."
        actions={
          <Button leftIcon="add" onClick={() => setOpen(true)}>
            New Service
          </Button>
        }
      />

      {isLoading && <div className="space-y-sm"><SkeletonList rows={4} /></div>}
      {!isLoading && !apps?.length && (
        <EmptyState
          icon="rocket_launch"
          title="No applications"
          description="Deploy OCI images or builds from git repositories, with health checks, rollback and domains."
          action={<Button onClick={() => setOpen(true)}>Create your first application</Button>}
        />
      )}
      {!!apps?.length || !!databases?.length ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-md">
          {(databases ?? []).map((db) => {
            const pill =
              db.status === "ready"
                ? { status: "running", pulse: true }
                : db.status === "creating" || db.status === "starting"
                  ? { status: "provisioning", pulse: true }
                  : db.status === "failed"
                    ? { status: "error", pulse: false }
                    : { status: db.status, pulse: false };
            return (
              <Link
                key={"db-" + db.id}
                to={"/databases/" + db.id}
                className="group bg-surface-container-low border border-outline-variant rounded-lg p-md flex flex-col gap-sm hover:border-primary/50 hover:bg-surface-container-high/40 transition-colors cursor-pointer min-w-0"
                title={`Open ${db.name}`}
              >
                <div className="flex items-start justify-between gap-sm">
                  <TechIcon name={db.engine} size={22} className="text-primary shrink-0" />
                  <StatusPill status={pill.status} pulse={pill.pulse} />
                </div>
                <div className="min-w-0">
                  <h3 className="font-body-md text-body-md text-on-surface font-semibold truncate">{db.name}</h3>
                  <p className="font-code-md text-code-md text-on-surface-variant/60 truncate">
                    {db.engine} {db.version}
                  </p>
                </div>
                <div className="flex items-center gap-sm mt-auto pt-sm border-t border-outline-variant/40 font-code-md text-code-md text-on-surface-variant/60">
                  <span className="inline-flex items-center gap-1">
                    <span className="material-symbols-outlined text-[14px]">counter_1</span>
                    :{db.port}
                  </span>
                  <span className="material-symbols-outlined text-[16px] text-on-surface-variant/30 group-hover:text-primary transition-colors ml-auto">chevron_right</span>
                </div>
              </Link>
            );
          })}
          {(apps ?? []).map((app) => {
            const state = states?.[app.id];
            const pill =
              state === "running"
                ? { status: "running", pulse: true }
                : state === "paused"
                  ? { status: "paused", pulse: false }
                  : state === "dead" || state === "error"
                    ? { status: "error", pulse: false }
                    : { status: "provisioning", pulse: true };
            return (
              <Link
                key={app.id}
                to="/apps/$appId"
                params={{ appId: app.id }}
                className="group bg-surface-container-low border border-outline-variant rounded-lg p-md flex flex-col gap-sm hover:border-primary/50 hover:bg-surface-container-high/40 transition-colors cursor-pointer min-w-0"
                title={`Open ${app.name}`}
              >
                <div className="flex items-start justify-between gap-sm">
                  <TechIcon name={app.source_type === "git" ? "gitlab" : "docker"} size={22} className="text-primary shrink-0" />
                  <StatusPill status={pill.status} pulse={pill.pulse} />
                </div>
                <div className="min-w-0">
                  <h3 className="font-body-md text-body-md text-on-surface font-semibold truncate">{app.name}</h3>
                  <p className="font-code-md text-code-md text-on-surface-variant/60 truncate">
                    {app.source_type === "image" ? app.image : app.git_url}
                  </p>
                </div>
                <div className="flex items-center gap-sm mt-auto pt-sm border-t border-outline-variant/40 font-code-md text-code-md text-on-surface-variant/60">
                  <span className="inline-flex items-center gap-1">
                    <span className="material-symbols-outlined text-[14px]">counter_1</span>
                    :{app.port}
                  </span>
                  {app.resources.mem_mb ? (
                    <span className="inline-flex items-center gap-1">
                      <span className="material-symbols-outlined text-[14px]">memory</span>
                      {app.resources.mem_mb >= 1024 ? `${app.resources.mem_mb / 1024} GB` : `${app.resources.mem_mb} MB`}
                    </span>
                  ) : null}
                  <span className="material-symbols-outlined text-[16px] text-on-surface-variant/30 group-hover:text-primary transition-colors ml-auto">chevron_right</span>
                </div>
              </Link>
            );
          })}
        </div>
      ) : null}

      <CreateServiceLauncher open={open} onClose={() => setOpen(false)} />
    </AppPage>
  );
}

export const Route = createFileRoute('/_shell/apps/')({
  component: Apps,
});
