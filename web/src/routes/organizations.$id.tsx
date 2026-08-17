import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { useOrg } from "../components/OrgProvider";
import { useOrgAudit, useOrgDetail, useOrgMembers, useProjects } from "../hooks";
import { apiDelete, apiPost, apiPut } from "../api/client";
import { useQueryClient } from "@tanstack/react-query";
import { useToast } from "../components/ui";
import type { OrgMember } from "../api/types";

export const Route = createFileRoute("/organizations/$id")({
  component: OrganizationPage,
});

const TABS = ["overview", "members", "projects", "audit"] as const;
type Tab = (typeof TABS)[number];

const ROLE_BADGE: Record<string, string> = {
  owner: "bg-[#7c3aed]/10 text-[#a78bfa] border-[#7c3aed]/30",
  admin: "bg-[#0ea5e9]/10 text-[#38bdf8] border-[#0ea5e9]/30",
  member: "bg-[#10b981]/10 text-[#34d399] border-[#10b981]/30",
  developer: "bg-[#10b981]/10 text-[#34d399] border-[#10b981]/30",
  viewer: "bg-surface-container-highest text-on-surface-variant border-outline-variant",
};

function timeAgo(iso: string): string {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

const ACTION_META: Record<string, { icon: string; color: string }> = {
  "project.created": { icon: "folder_copy", color: "text-[#34d399]" },
  "project.deleted": { icon: "folder_off", color: "text-error" },
  "member.invited": { icon: "person_add", color: "text-[#38bdf8]" },
  "member.removed": { icon: "person_remove", color: "text-error" },
  "member.role_changed": { icon: "manage_accounts", color: "text-[#fbbf24]" },
  "org.created": { icon: "add_business", color: "text-[#a78bfa]" },
  "org.updated": { icon: "edit", color: "text-[#fbbf24]" },
  "org.deleted": { icon: "delete", color: "text-error" },
  "deployment.ready": { icon: "rocket_launch", color: "text-[#34d399]" },
  "deployment.failed": { icon: "error", color: "text-error" },
};

function OrganizationPage() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { toast } = useToast();
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
  const canManage = role === "owner" || role === "admin" || role === "global:admin";

  const filteredMembers = useMemo(
    () => (members ?? []).filter((m) => !query || m.name.toLowerCase().includes(query.toLowerCase()) || m.email.toLowerCase().includes(query.toLowerCase())),
    [members, query]
  );

  const invite = async () => {
    if (!email.trim()) {
      toast("Email is required", "error");
      return;
    }
    try {
      await apiPost(`/api/v1/organizations/${id}/members`, {
        email,
        role: invRole,
        projects: [...projSel],
      });
      toast("Member invited");
      setInviteOpen(false);
      setEmail("");
      setProjSel(new Set());
      qc.invalidateQueries({ queryKey: ["org-members"] });
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to invite", "error");
    }
  };

  const setRole = async (m: OrgMember, newRole: string) => {
    try {
      await apiPut(`/api/v1/organizations/${id}/members/${m.user_id}`, { role: newRole });
      toast(`Role updated to ${newRole}`);
      qc.invalidateQueries({ queryKey: ["org-members"] });
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
    }
  };

  const removeMember = async (m: OrgMember) => {
    try {
      await apiDelete(`/api/v1/organizations/${id}/members/${m.user_id}`);
      qc.invalidateQueries({ queryKey: ["org-members"] });
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed", "error");
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

  const avatarBg = org?.color || "#7c3aed";

  return (
    <div className="max-w-[1200px] mx-auto">
      <div className="rounded-2xl border border-outline-variant bg-surface-container-lowest shadow-2xl overflow-hidden mb-lg">
        <div className="h-20" style={{ background: `linear-gradient(120deg, ${avatarBg}33, transparent)` }} />
        <div className="px-lg pb-lg -mt-10 flex flex-wrap items-end gap-lg">
          <div className="w-20 h-20 rounded-2xl border-4 border-surface-container-lowest flex items-center justify-center text-on-primary text-3xl font-bold shadow-xl shrink-0" style={{ background: avatarBg }}>
            {(org?.name || "?").slice(0, 2).toUpperCase()}
          </div>
          <div className="pb-sm flex-1 min-w-0">
            <div className="flex items-center gap-sm flex-wrap">
              <h1 className="font-headline-sm text-headline-sm font-bold text-on-surface">{org?.name ?? "Loading..."}</h1>
              <span className={`px-2 py-0.5 rounded-full border font-label-caps text-[10px] uppercase ${ROLE_BADGE[role] ?? ROLE_BADGE.member}`}>{role}</span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-xs">{org?.description || "No description."}</p>
          </div>
          <button
            onClick={() => {
              switchOrg(id);
              navigate({ to: "/projects" });
            }}
            className="px-md py-2 rounded-lg bg-primary text-on-primary font-body-sm font-semibold hover:bg-primary-fixed-dim transition-colors"
          >
            Open Workspace
          </button>
        </div>
      </div>

      <div className="flex items-center gap-sm border-b border-outline-variant mb-lg">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-md py-2.5 font-label-caps text-label-caps uppercase border-b-2 -mb-px transition-colors ${tab === t ? "border-primary text-primary" : "border-transparent text-on-surface-variant hover:text-on-surface"}`}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "overview" && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-md">
          <StatCard icon="folder_copy" label="Projects" value={projects?.length ?? 0} color="#34d399" />
          <StatCard icon="group" label="Members" value={members?.length ?? 0} color="#38bdf8" />
          <StatCard icon="receipt_long" label="Audit events" value={audit?.length ?? 0} color="#a78bfa" />
          <StatCard icon="vpn_key" label="Your role" value={role} color="#fbbf24" />
        </div>
      )}

      {tab === "members" && (
        <div>
          <div className="flex items-center justify-between mb-md">
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">Members</h2>
            {canManage && (
              <button
                onClick={() => setInviteOpen(true)}
                className="px-md py-1.5 rounded-lg bg-primary text-on-primary font-body-sm font-semibold hover:bg-primary-fixed-dim transition-colors"
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
                  <tr key={m.user_id} className="border-b border-outline-variant/40 hover:bg-surface-container-high transition-colors">
                    <td className="px-md py-2.5">
                      <div className="flex items-center gap-sm">
                        <div className="w-8 h-8 rounded-full bg-primary-container flex items-center justify-center text-on-primary-container text-[12px] font-bold">
                          {m.name.slice(0, 2).toUpperCase()}
                        </div>
                        <div>
                          <p className="font-body-sm font-semibold text-on-surface">{m.name}</p>
                          <p className="font-code-md text-[11px] text-on-surface-variant">{m.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-md py-2.5">
                      {canManage ? (
                        <select
                          value={m.role === "developer" ? "member" : m.role}
                          onChange={(e) => setRole(m, e.target.value)}
                          className="bg-surface-container-lowest border border-outline-variant rounded px-2 py-1 font-code-md text-code-md text-on-surface focus:border-primary focus:outline-none"
                        >
                          <option value="admin">admin</option>
                          <option value="member">member</option>
                          <option value="viewer">viewer</option>
                        </select>
                      ) : (
                        <span className={`px-2 py-0.5 rounded-full border font-label-caps text-[10px] uppercase ${ROLE_BADGE[m.role] ?? ROLE_BADGE.member}`}>{m.role}</span>
                      )}
                    </td>
                    <td className="px-md py-2.5">
                      <span className="font-code-md text-code-md text-on-surface-variant">
                        {m.projects?.length > 0 ? `${m.projects.length} project${m.projects.length > 1 ? "s" : ""}` : "all projects"}
                      </span>
                    </td>
                    <td className="px-md py-2.5 text-right">
                      {canManage && m.role !== "owner" && (
                        <button onClick={() => removeMember(m)} className="text-on-surface-variant hover:text-error transition-colors px-1">
                          <span className="material-symbols-outlined text-[18px]">person_remove</span>
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
                {filteredMembers.length === 0 && (
                  <tr>
                    <td colSpan={4} className="px-md py-lg text-center text-on-surface-variant">No members yet. Invite your teammates.</td>
                  </tr>
                )}
              </tbody>
            </table>
            </div>
          </div>
          </div>
        )}

      {tab === "projects" && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-md">
          {(projects ?? []).map((p) => (
            <button
              key={p.id}
              onClick={() => navigate({ to: "/projects/$projectId", params: { projectId: p.id } })}
              className="rounded-xl border border-outline-variant bg-surface-container-lowest p-md text-left hover:border-primary/40 hover:shadow-lg transition-all"
            >
              <div className="w-10 h-10 rounded-lg flex items-center justify-center mb-sm" style={{ background: `${p.color || "#7c3aed"}22`, color: p.color || "#7c3aed" }}>
                <span className="material-symbols-outlined">folder_copy</span>
              </div>
              <p className="font-body-md font-semibold text-on-surface">{p.name}</p>
              <p className="font-code-md text-[11px] text-on-surface-variant">{p.slug}</p>
            </button>
          ))}
          {projects?.length === 0 && (
            <div className="col-span-full rounded-xl border border-dashed border-outline-variant p-xl text-center">
              <p className="font-body-sm text-on-surface-variant">No projects in this organization yet.</p>
              <button onClick={() => navigate({ to: "/projects/new" })} className="mt-md px-md py-2 rounded-lg bg-primary text-on-primary font-body-sm font-semibold">
                Create your first project
              </button>
            </div>
          )}
        </div>
      )}

      {tab === "audit" && (
        <div className="rounded-xl border border-outline-variant bg-surface-container-lowest overflow-hidden">
          {(audit ?? []).map((a) => {
            const meta = ACTION_META[a.action] ?? { icon: "history", color: "text-on-surface-variant" };
            return (
              <div key={a.id} className="flex items-start gap-sm px-md py-2.5 border-b border-outline-variant/40 hover:bg-surface-container-high transition-colors">
                <span className={`material-symbols-outlined text-[18px] mt-0.5 ${meta.color}`}>{meta.icon}</span>
                <div className="flex-1 min-w-0">
                  <p className="font-body-sm text-on-surface">{a.action.replace(/[._]/g, " ")} <span className="text-on-surface-variant">· {a.details}</span></p>
                  <p className="font-code-md text-[11px] text-on-surface-variant/60">{a.resource_type} · {timeAgo(a.created_at)}</p>
                </div>
              </div>
            );
          })}
          {audit?.length === 0 && <p className="p-md text-center text-on-surface-variant">No audit events yet.</p>}
        </div>
      )}

      {inviteOpen && (
        <div className="fixed inset-0 z-[80] flex items-center justify-center bg-black/60 p-4" onClick={() => setInviteOpen(false)}>
          <div
            onClick={(e) => e.stopPropagation()}
            className="bg-surface-container-lowest border border-outline-variant rounded-2xl w-full max-w-[28rem] p-lg shadow-2xl animate-modal-pop"
          >
            <h2 className="font-headline-sm text-headline-sm font-bold text-on-surface mb-xs">Invite Member</h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-lg">They'll be able to access the projects you assign.</p>
            <div className="flex flex-col gap-md">
              <div className="flex flex-col gap-xs">
                <label className="font-label-caps text-label-caps text-on-surface-variant">Email</label>
                <input
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="teammate@company.com"
                  className="bg-surface-container-lowest border border-outline-variant rounded-lg px-sm py-2 font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant/50 focus:border-primary focus:outline-none"
                />
              </div>
              <div className="flex flex-col gap-xs">
                <label className="font-label-caps text-label-caps text-on-surface-variant">Role</label>
                <select
                  value={invRole}
                  onChange={(e) => setInvRole(e.target.value)}
                  className="bg-surface-container-lowest border border-outline-variant rounded-lg px-sm py-2 font-body-md text-body-md text-on-surface focus:border-primary focus:outline-none"
                >
                  <option value="member">Member</option>
                  <option value="admin">Admin</option>
                  <option value="viewer">Viewer</option>
                </select>
              </div>
              <div className="flex flex-col gap-xs">
                <label className="font-label-caps text-label-caps text-on-surface-variant">Assign projects</label>
                <div className="max-h-40 overflow-y-auto sidebar-scroll border border-outline-variant rounded-lg divide-y divide-outline-variant/40">
                  {(projects ?? []).map((p) => (
                    <label key={p.id} className="flex items-center gap-sm px-sm py-2 cursor-pointer hover:bg-surface-container-high">
                      <input
                        type="checkbox"
                        checked={projSel.has(p.id)}
                        onChange={() => toggleProject(p.id)}
                        className="w-4 h-4 rounded-sm bg-surface border-outline-variant text-primary"
                      />
                      <span className="font-body-sm text-on-surface">{p.name}</span>
                    </label>
                  ))}
                  {projects?.length === 0 && <p className="p-sm text-on-surface-variant font-body-sm">No projects yet.</p>}
                </div>
              </div>
              <div className="flex justify-end gap-sm pt-md border-t border-outline-variant">
                <button onClick={() => setInviteOpen(false)} className="px-md py-2 rounded-lg text-on-surface-variant hover:text-on-surface font-body-sm">
                  Cancel
                </button>
                <button onClick={invite} className="px-md py-2 rounded-lg bg-primary text-on-primary font-body-sm font-semibold hover:bg-primary-fixed-dim transition-colors">
                  Send Invite
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({ icon, label, value, color }: { icon: string; label: string; value: string | number; color: string }) {
  return (
    <div className="rounded-xl border border-outline-variant bg-surface-container-lowest p-md">
      <span className={`material-symbols-outlined text-2xl ${color}`} style={{ color }}>{icon}</span>
      <p className="font-headline-md text-headline-md font-bold text-on-surface mt-sm">{value}</p>
      <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">{label}</p>
    </div>
  );
}
