import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo, useState, type KeyboardEvent } from "react";
import {
  Archive,
  Bell,
  Briefcase,
  FolderOpen,
  Gear,
  Person,
  PersonSimpleRun,
  RocketLaunch,
  Trash,
  UserPlus,
  Warning,
  type Icon,
} from "@phosphor-icons/react";
import { useOrg } from "../components/OrgProvider";
import {
  useOrgAudit,
  useOrgDetail,
  useOrgMembers,
  useProjects,
} from "../hooks";
import { apiDelete, apiPost, apiPut } from "../api/client";
import { useQueryClient } from "@tanstack/react-query";
import {
  Badge,
  Button,
  Checkbox,
  EmptyState,
  Field,
  Input,
  Modal,
  NativeSelect,
  Skeleton,
  useToast,
} from "@aether/design-system";
import type { OrgMember } from "../api/types";

export const Route = createFileRoute("/organizations/$id")({
  component: OrganizationPage,
});

const TABS = ["overview", "members", "projects", "audit"] as const;
type Tab = (typeof TABS)[number];

const ROLE_BADGE: Record<string, string> = {
  owner: "bg-primary/10 text-primary border-primary/30",
  admin: "bg-status-info-container/20 text-status-info border-status-info/30",
  member:
    "bg-status-success-container/20 text-status-success border-status-success/30",
  developer:
    "bg-status-success-container/20 text-status-success border-status-success/30",
  viewer:
    "bg-surface-container-highest text-on-surface-variant border-outline-variant",
};

function timeAgo(iso: string): string {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

const ACTION_META: Record<string, { icon: Icon; color: string }> = {
  "project.created": { icon: FolderOpen, color: "text-status-success" },
  "project.deleted": { icon: Trash, color: "text-error" },
  "member.invited": { icon: UserPlus, color: "text-status-info" },
  "member.removed": { icon: PersonSimpleRun, color: "text-error" },
  "member.role_changed": { icon: Gear, color: "text-status-warning" },
  "org.created": { icon: Briefcase, color: "text-secondary" },
  "org.updated": { icon: Gear, color: "text-status-warning" },
  "org.deleted": { icon: Trash, color: "text-error" },
  "deployment.ready": { icon: RocketLaunch, color: "text-status-success" },
  "deployment.failed": { icon: Warning, color: "text-error" },
};

function OrganizationPage() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { add } = useToast();
  const { currentOrg, role, switchOrg } = useOrg();
  const { data: org } = useOrgDetail(id);
  const { data: members } = useOrgMembers(id);
  const { data: audit } = useOrgAudit(id);
  const { data: projects } = useProjects();
  const [tab, setTab] = useState<Tab>("overview");
  const [inviteOpen, setInviteOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [invRole, setInvRole] = useState("member");
  const [projSel, setProjSel] = useState<Set<string>>(new Set());
  const [query, setQuery] = useState("");
  const canManage =
    role === "owner" || role === "admin" || role === "global:admin";

  const filteredMembers = useMemo(
    () =>
      (members ?? []).filter(
        (m) =>
          !query ||
          m.name.toLowerCase().includes(query.toLowerCase()) ||
          m.email.toLowerCase().includes(query.toLowerCase()),
      ),
    [members, query],
  );

  const invite = async () => {
    if (!email.trim()) {
      add({ title: "Email is required", tone: "error" });
      return;
    }
    try {
      await apiPost(`/api/v1/organizations/${id}/members`, {
        email,
        role: invRole,
        projects: [...projSel],
      });
      add({ title: "Member invited", tone: "success" });
      setInviteOpen(false);
      setEmail("");
      setProjSel(new Set());
      qc.invalidateQueries({ queryKey: ["org-members"] });
    } catch (err) {
      add({
        title: err instanceof Error ? err.message : "Failed to invite",
        tone: "error",
      });
    }
  };

  const setRole = async (m: OrgMember, newRole: string) => {
    try {
      await apiPut(`/api/v1/organizations/${id}/members/${m.user_id}`, {
        role: newRole,
      });
      add({ title: `Role updated to ${newRole}`, tone: "success" });
      qc.invalidateQueries({ queryKey: ["org-members"] });
    } catch (err) {
      add({
        title: err instanceof Error ? err.message : "Failed to update role",
        tone: "error",
      });
    }
  };

  const removeMember = async (m: OrgMember) => {
    try {
      await apiDelete(`/api/v1/organizations/${id}/members/${m.user_id}`);
      qc.invalidateQueries({ queryKey: ["org-members"] });
    } catch (err) {
      add({
        title: err instanceof Error ? err.message : "Failed to remove member",
        tone: "error",
      });
    }
  };

  const toggleProject = (pid: string) => {
    setProjSel((prev) => {
      const next = new Set(prev);
      if (next.has(pid)) next.delete(pid);
      else next.add(pid);
      return next;
    });
  };

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
          ? TABS.length - 1
          : (index + (event.key === "ArrowRight" ? 1 : -1) + TABS.length) %
            TABS.length;
    const nextTab = TABS[nextIndex];
    setTab(nextTab);
    document.getElementById(`organization-tab-${nextTab}`)?.focus();
  };

  const avatarBg = org?.color || "#7c3aed";

  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-lg p-6 lg:p-8">
      <div className="overflow-hidden rounded-2xl border border-border bg-gradient-to-br from-primary/10 via-surface-card to-secondary/5 shadow-md">
        <div
          className="h-20 opacity-70"
          style={{
            background: `linear-gradient(120deg, ${avatarBg}33, transparent)`,
          }}
        />
        <div className="-mt-10 flex flex-wrap items-end gap-lg px-lg pb-lg">
          <div
            className="flex size-20 shrink-0 items-center justify-center rounded-2xl border-4 border-surface-card text-3xl font-bold text-on-primary shadow-md"
            style={{ background: avatarBg }}
          >
            {(org?.name || "?").slice(0, 2).toUpperCase()}
          </div>
          <div className="basis-full pb-sm sm:min-w-0 sm:flex-1">
            <div className="flex flex-wrap items-center gap-sm">
              <h1 className="min-w-0 max-w-full break-words font-display-lg text-4xl tracking-[-0.04em] text-on-surface sm:text-display-lg">
                {org ? (
                  org.name
                ) : (
                  <span className="inline-block w-48">
                    <Skeleton variant="text" />
                  </span>
                )}
              </h1>
              <span
                className={`rounded-full border px-2 py-0.5 font-label-caps text-[10px] uppercase ${ROLE_BADGE[role] ?? ROLE_BADGE.member}`}
              >
                {role}
              </span>
            </div>
            <p className="mt-xs text-body-md text-on-surface-variant">
              {org?.description || "No description."}
            </p>
          </div>
          <button
            onClick={() => {
              switchOrg(id);
              navigate({ to: "/projects" });
            }}
            type="button"
            className="rounded-lg bg-primary px-md py-2 font-body-sm font-semibold text-on-primary outline-none transition-[background-color,box-shadow,transform] duration-150 hover:bg-primary-fixed-dim active:scale-[0.985] focus-visible:ring-2 focus-visible:ring-ring"
          >
            Open Workspace
          </button>
        </div>
      </div>

      <div
        className="mb-lg flex flex-wrap items-center gap-1 rounded-xl border border-border bg-surface-container p-1"
        role="tablist"
        aria-label="Organization views"
      >
        {TABS.map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            role="tab"
            id={`organization-tab-${t}`}
            aria-controls={`organization-panel-${t}`}
            aria-selected={tab === t}
            tabIndex={tab === t ? 0 : -1}
            onKeyDown={(event) => handleTabKeyDown(event, TABS.indexOf(t))}
            className={`rounded-lg px-md py-2.5 font-label-caps text-label-caps uppercase outline-none transition-[background-color,border-color,color,transform] duration-150 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] ${tab === t ? "bg-surface-card text-primary shadow-sm" : "text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface"}`}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "overview" && (
        <div
          id="organization-panel-overview"
          role="tabpanel"
          aria-labelledby="organization-tab-overview"
          tabIndex={0}
          className="grid grid-cols-1 md:grid-cols-4 gap-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <StatCard
            icon={FolderOpen}
            label="Projects"
            value={projects?.length ?? 0}
            color="text-status-success"
          />
          <StatCard
            icon={Person}
            label="Members"
            value={members?.length ?? 0}
            color="text-status-info"
          />
          <StatCard
            icon={Bell}
            label="Audit events"
            value={audit?.length ?? 0}
            color="text-secondary"
          />
          <StatCard
            icon={Gear}
            label="Your role"
            value={role}
            color="text-status-warning"
          />
        </div>
      )}

      {tab === "members" && (
        <div
          id="organization-panel-members"
          role="tabpanel"
          aria-labelledby="organization-tab-members"
          tabIndex={0}
          className="outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <div className="flex items-center justify-between mb-md">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">
              Members
            </h2>
            {canManage && (
              <button
                onClick={() => setInviteOpen(true)}
                type="button"
                className="rounded-lg bg-primary px-md py-1.5 font-body-sm font-semibold text-on-primary outline-none transition-[background-color,box-shadow,transform] duration-150 hover:bg-primary-fixed-dim active:scale-[0.985] focus-visible:ring-2 focus-visible:ring-ring"
              >
                Invite Member
              </button>
            )}
          </div>
          <div className="rounded-xl border border-outline-variant overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead>
                  <tr className="border-b border-outline-variant font-label-caps text-label-caps text-on-surface-variant/60 uppercase bg-surface-container-low">
                    <th className="px-md py-2.5">Member</th>
                    <th className="px-md py-2.5">Role</th>
                    <th className="px-md py-2.5">Assigned Projects</th>
                    <th className="px-md py-2.5 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredMembers.map((m) => (
                    <tr
                      key={m.user_id}
                      className="border-b border-outline-variant/40 hover:bg-surface-container-high transition-colors"
                    >
                      <td className="px-md py-2.5">
                        <div className="flex items-center gap-sm">
                          <div className="w-8 h-8 rounded-full bg-primary-container flex items-center justify-center text-on-primary-container text-[12px] font-bold">
                            {m.name.slice(0, 2).toUpperCase()}
                          </div>
                          <div>
                            <p className="font-body-sm font-semibold text-on-surface">
                              {m.name}
                            </p>
                            <p className="font-code-md text-[11px] text-on-surface-variant">
                              {m.email}
                            </p>
                          </div>
                        </div>
                      </td>
                      <td className="px-md py-2.5">
                        {canManage ? (
                          <NativeSelect
                            value={m.role === "developer" ? "member" : m.role}
                            onChange={(event) => setRole(m, event.target.value)}
                            options={[
                              { label: "admin", value: "admin" },
                              { label: "member", value: "member" },
                              { label: "viewer", value: "viewer" },
                            ]}
                          />
                        ) : (
                          <span
                            className={`px-2 py-0.5 rounded-full border font-label-caps text-[10px] uppercase ${ROLE_BADGE[m.role] ?? ROLE_BADGE.member}`}
                          >
                            {m.role}
                          </span>
                        )}
                      </td>
                      <td className="px-md py-2.5">
                        <span className="font-code-md text-code-md text-on-surface-variant">
                          {m.projects?.length > 0
                            ? `${m.projects.length} project${m.projects.length > 1 ? "s" : ""}`
                            : "all projects"}
                        </span>
                      </td>
                      <td className="px-md py-2.5 text-right">
                        {canManage && m.role !== "owner" && (
                          <button
                            type="button"
                            onClick={() => removeMember(m)}
                            aria-label={`Remove ${m.name}`}
                            className="rounded p-1 text-on-surface-variant outline-none transition-[background-color,color] duration-150 hover:bg-error/10 hover:text-error focus-visible:ring-2 focus-visible:ring-ring"
                          >
                            <PersonSimpleRun size={18} aria-hidden="true" />
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                  {filteredMembers.length === 0 && (
                    <tr>
                      <td colSpan={4}>
                        <EmptyState
                          title="No members yet"
                          description="Invite your teammates."
                          className="border-0"
                        />
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {tab === "projects" && (
        <div
          id="organization-panel-projects"
          role="tabpanel"
          aria-labelledby="organization-tab-projects"
          tabIndex={0}
          className="grid grid-cols-1 md:grid-cols-3 gap-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {(projects ?? []).map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() =>
                navigate({
                  to: "/projects/$projectId",
                  params: { projectId: p.id },
                })
              }
              className="rounded-xl border border-outline-variant bg-surface-container-lowest p-md text-left outline-none transition-[background-color,border-color,box-shadow,transform] duration-200 hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md active:translate-y-0 focus-visible:ring-2 focus-visible:ring-ring"
            >
              <div
                className="w-10 h-10 rounded-lg flex items-center justify-center mb-sm"
                style={{
                  background: `${p.color || "#7c3aed"}22`,
                  color: p.color || "#7c3aed",
                }}
              >
                <FolderOpen size={22} aria-hidden="true" />
              </div>
              <p className="font-body-md font-semibold text-on-surface">
                {p.name}
              </p>
              <p className="font-code-md text-[11px] text-on-surface-variant">
                {p.slug}
              </p>
            </button>
          ))}
          {projects?.length === 0 && (
            <div className="col-span-full">
              <EmptyState
                title="No projects in this organization yet"
                action={
                  <Button onClick={() => navigate({ to: "/projects/new" })}>
                    Create your first project
                  </Button>
                }
              />
            </div>
          )}
        </div>
      )}

      {tab === "audit" && (
        <div
          id="organization-panel-audit"
          role="tabpanel"
          aria-labelledby="organization-tab-audit"
          tabIndex={0}
          className="rounded-xl border border-outline-variant bg-surface-container-lowest overflow-hidden outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {(audit ?? []).map((a) => {
            const meta = ACTION_META[a.action] ?? {
              icon: Archive,
              color: "text-on-surface-variant",
            };
            return (
              <div
                key={a.id}
                className="flex items-start gap-sm px-md py-2.5 border-b border-outline-variant/40 hover:bg-surface-container-high transition-colors"
              >
                <meta.icon
                  size={18}
                  className={`mt-0.5 ${meta.color}`}
                  aria-hidden="true"
                />
                <div className="flex-1 min-w-0">
                  <p className="font-body-sm text-on-surface">
                    {a.action.replace(/[._]/g, " ")}{" "}
                    <span className="text-on-surface-variant">
                      · {a.details}
                    </span>
                  </p>
                  <p className="font-code-md text-[11px] text-on-surface-variant/60">
                    {a.resource_type} · {timeAgo(a.created_at)}
                  </p>
                </div>
              </div>
            );
          })}
          {audit?.length === 0 && (
            <EmptyState title="No audit events yet" className="border-0" />
          )}
        </div>
      )}

      <Modal
        open={inviteOpen}
        onOpenChange={(value) => setInviteOpen(value)}
        title="Invite Member"
        description="They'll be able to access the projects you assign."
        size="sm"
      >
        <div className="space-y-md">
          <div className="flex flex-col gap-md">
            <Input
              label="Email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="teammate@company.com"
            />
            <NativeSelect
              label="Role"
              value={invRole}
              onChange={(event) => setInvRole(event.target.value)}
              options={[
                { label: "Member", value: "member" },
                { label: "Admin", value: "admin" },
                { label: "Viewer", value: "viewer" },
              ]}
            />
            <div className="flex flex-col gap-xs">
              <p className="font-label-caps text-label-caps text-on-surface-variant">
                Assign projects
              </p>
              <div className="max-h-40 overflow-y-auto sidebar-scroll border border-outline-variant rounded-lg divide-y divide-outline-variant/40">
                {(projects ?? []).map((p) => (
                  <div
                    key={p.id}
                    className="flex items-center gap-sm px-sm py-2 hover:bg-surface-container-high"
                  >
                    <Checkbox
                      checked={projSel.has(p.id)}
                      onCheckedChange={() => toggleProject(p.id)}
                    />
                    <span className="font-body-sm text-on-surface">
                      {p.name}
                    </span>
                  </div>
                ))}
                {projects?.length === 0 && (
                  <EmptyState
                    title="No projects yet"
                    className="border-0 p-4"
                  />
                )}
              </div>
            </div>
            <div className="flex justify-end gap-sm pt-md border-t border-outline-variant">
              <Button variant="ghost" onClick={() => setInviteOpen(false)}>
                Cancel
              </Button>
              <Button onClick={invite}>Send Invite</Button>
            </div>
          </div>
        </div>
      </Modal>
    </main>
  );
}

function StatCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: Icon;
  label: string;
  value: string | number;
  color: string;
}) {
  return (
    <div className="rounded-xl border border-outline-variant bg-surface-container-lowest p-md">
      <Icon size={28} className={color} aria-hidden="true" />
      <p className="font-headline-md text-headline-md font-bold text-on-surface mt-sm">
        {value}
      </p>
      <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">
        {label}
      </p>
    </div>
  );
}
