import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useOrg } from "../components/OrgProvider";
import { useToast } from "../components/ui";
import { apiPost } from "../api/client";

export const Route = createFileRoute("/organizations/new")({
  component: CreateOrganization,
});

function CreateOrganization() {
  const { refetch } = useOrg();
  const { toast } = useToast();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [color, setColor] = useState("#7c3aed");
  const [creating, setCreating] = useState(false);

  const create = async () => {
    if (!name.trim()) {
      toast("Organization name is required", "error");
      return;
    }
    setCreating(true);
    try {
      const org = await apiPost<{ id: string }>("/api/v1/organizations", { name, description, color });
      await refetch();
      toast("Organization created");
      window.location.href = `/organizations/${org.id}`;
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create organization", "error");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="max-w-[32rem] mx-auto py-xl">
      <div className="rounded-2xl border border-outline-variant bg-surface-container-lowest shadow-2xl overflow-hidden">
        <div className="p-lg border-b border-outline-variant">
          <h1 className="font-headline-sm text-headline-sm font-bold text-on-surface">Create Organization</h1>
          <p className="font-body-sm text-body-sm text-on-surface-variant mt-xs">Create a workspace for your team — projects and services will be scoped to it.</p>
        </div>
        <div className="p-lg space-y-lg">
          <div className="flex items-center gap-lg">
            <div className="w-16 h-16 rounded-xl flex items-center justify-center text-on-primary text-2xl font-bold" style={{ background: color }}>
              {name ? name.slice(0, 2).toUpperCase() : "?"}
            </div>
            <div className="flex gap-sm">
              {["#7c3aed", "#0ea5e9", "#10b981", "#f59e0b", "#ef4444", "#ec4899"].map((c) => (
                <button
                  key={c}
                  onClick={() => setColor(c)}
                  className={`w-7 h-7 rounded-full transition-transform ${color === c ? "ring-2 ring-offset-2 ring-primary scale-110" : "hover:scale-110"}`}
                  style={{ background: c }}
                />
              ))}
            </div>
          </div>
          <div className="flex flex-col gap-xs">
            <label className="font-label-caps text-label-caps text-on-surface-variant">Name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Acme Corp"
              className="bg-surface-container-lowest border border-outline-variant rounded-lg px-sm py-2 font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant/50 focus:border-primary focus:outline-none"
            />
          </div>
          <div className="flex flex-col gap-xs">
            <label className="font-label-caps text-label-caps text-on-surface-variant">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description"
              rows={2}
              className="bg-surface-container-lowest border border-outline-variant rounded-lg px-sm py-2 font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant/50 focus:border-primary focus:outline-none resize-none"
            />
          </div>
          <button
            onClick={create}
            disabled={creating}
            className="w-full py-2.5 rounded-lg bg-primary text-on-primary font-body-md font-semibold hover:bg-primary-fixed-dim transition-colors disabled:opacity-50"
          >
            {creating ? "Creating..." : "Create Organization"}
          </button>
        </div>
      </div>
    </div>
  );
}
