import { useState } from "react";
import { useCreateCompose, useProjects } from "../api/hooks";
import { useOverlayGate } from "./OverlayManager";
import { ComposeEditor } from "./ComposeEditor";
import { Button, Input, Select, useToast } from "./ui";

const DEFAULT_COMPOSE = `services:
  app:
    image: nginx:alpine
    ports:
      - "80:80"
    restart: unless-stopped`;

export function ComposeWizard({ open, onClose, fixedProjectId }: { open: boolean; onClose: () => void; fixedProjectId?: string }) {
  const { data: projects } = useProjects();
  const createCompose = useCreateCompose();
  const { toast } = useToast();
  const [projectId, setProjectId] = useState(fixedProjectId ?? "");
  const [name, setName] = useState("");
  const [content, setContent] = useState(DEFAULT_COMPOSE);
  const [creating, setCreating] = useState(false);
  const { mounted, closing, close } = useOverlayGate("compose-wizard", open, onClose);

  if (!mounted) return null;

  const create = async () => {
    if (!projectId || !name.trim()) {
      toast("Project and name are required", "error");
      return;
    }
    setCreating(true);
    try {
      await createCompose.mutateAsync({ project_id: projectId, name, content });
      toast(`Compose stack "${name}" created`);
      onClose();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create stack", "error");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className={`fixed inset-0 z-[80] flex items-center justify-center bg-black/60 p-4 ${closing ? "animate-fade-out" : "animate-fade-in"}`} onClick={() => close()}>
      <div role="dialog" aria-modal="true" aria-label="Compose wizard" onClick={(e) => e.stopPropagation()} className="w-full max-w-4xl bg-surface-modal border border-outline-variant rounded-2xl shadow-xl animate-modal-pop overflow-hidden">
        <div className="flex items-center justify-between px-xl pt-xl pb-md border-b border-outline-variant">
          <div>
            <h2 className="font-headline-sm text-headline-sm text-on-surface">🐳 Docker Compose · Create</h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant">Live validation, dependency graph and preview on the right.</p>
          </div>
          <button onClick={() => close()} className="material-symbols-outlined text-on-surface-variant hover:text-on-surface transition-colors">close</button>
        </div>

        <div className="p-xl space-y-lg max-h-[70vh] overflow-y-auto sidebar-scroll">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
            <div>
              <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Project</p>
              <Select value={projectId} onChange={(e) => setProjectId(e.target.value)} disabled={!!fixedProjectId}>
                <option value="">Select...</option>
                {(projects ?? []).map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </Select>
            </div>
            <div>
              <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Stack name</p>
              <Input icon="label" placeholder="my-stack" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
          </div>
          <ComposeEditor value={content} onChange={setContent} />
        </div>

        <div className="flex justify-between px-xl py-lg border-t border-outline-variant">
          <Button variant="ghost" onClick={() => close()}>Cancel</Button>
          <Button onClick={create} disabled={creating}>
            {creating ? "Creating..." : "Create stack"}
          </Button>
        </div>
      </div>
    </div>
  );
}
