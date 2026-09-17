import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useOrg } from "../components/OrgProvider";
import {
  Button,
  Field,
  Input,
  Textarea,
  useToast,
} from "@aether/design-system";
import { apiPost } from "../api/client";

export const Route = createFileRoute("/organizations/new")({
  component: CreateOrganization,
});

function CreateOrganization() {
  const { refetch } = useOrg();
  const { add } = useToast();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [color, setColor] = useState("#7c3aed");
  const [creating, setCreating] = useState(false);

  const create = async () => {
    if (!name.trim()) {
      add({ title: "Organization name is required", tone: "error" });
      return;
    }
    setCreating(true);
    try {
      const org = await apiPost<{ id: string }>("/api/v1/organizations", {
        name,
        description,
        color,
      });
      await refetch();
      window.location.href = `/organizations/${org.id}`;
    } catch (err) {
      add({
        title:
          err instanceof Error ? err.message : "Failed to create organization",
        tone: "error",
      });
    } finally {
      setCreating(false);
    }
  };

  return (
    <main className="mx-auto flex min-h-full w-full max-w-3xl items-center justify-center p-6 lg:p-8">
      <div className="w-full overflow-hidden rounded-2xl border border-border bg-gradient-to-br from-primary/10 via-surface-card to-secondary/5 shadow-md">
        <header className="border-b border-border px-lg py-xl">
          <p className="font-label-caps text-label-caps uppercase text-primary">
            Workspace setup
          </p>
          <h1 className="mt-sm font-display-lg text-display-lg tracking-[-0.04em] text-on-surface">
            Create organization
          </h1>
          <p className="mt-sm max-w-xl text-body-md text-on-surface-variant">
            Create a workspace for your team. Projects and services will be
            scoped to it.
          </p>
        </header>
        <div className="space-y-lg bg-surface-card p-lg sm:p-xl">
          <div className="flex flex-wrap items-center gap-lg">
            <div
              className="flex size-16 shrink-0 items-center justify-center rounded-2xl text-2xl font-bold text-on-primary shadow-md"
              style={{ background: color }}
            >
              {name ? name.slice(0, 2).toUpperCase() : "?"}
            </div>
            <div
              className="flex flex-wrap gap-sm"
              aria-label="Organization color"
            >
              {[
                "#7c3aed",
                "#0ea5e9",
                "#10b981",
                "#f59e0b",
                "#ef4444",
                "#ec4899",
              ].map((c) => (
                <button
                  key={c}
                  type="button"
                  aria-label={`Use ${c} as organization color`}
                  aria-pressed={color === c}
                  onClick={() => setColor(c)}
                  className={`size-8 rounded-full transition-transform focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${color === c ? "scale-110 ring-2 ring-primary ring-offset-2" : "hover:scale-110"}`}
                  style={{ background: c }}
                />
              ))}
            </div>
          </div>
          <Field label="Name">
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Acme Corp"
            />
          </Field>
          <Field
            label="Description"
            description="Optional context for teammates."
          >
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What is this workspace for?"
              rows={3}
            />
          </Field>
          <Button fullWidth onClick={create} disabled={creating}>
            {creating ? "Creating organization" : "Create organization"}
          </Button>
        </div>
      </div>
    </main>
  );
}
