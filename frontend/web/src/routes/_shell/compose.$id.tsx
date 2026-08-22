import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import Editor from "@monaco-editor/react";
import { AlertDialog, Badge, Button, Card, Skeleton, useToast } from "@aether/design-system";
import { FileCode, Play, Stop, Trash } from "@phosphor-icons/react";
import { useComposeStack, useComposeUp, useComposeDown, useDeleteCompose } from "../../hooks";

export const Route = createFileRoute("/_shell/compose/$id")({
  component: ComposeStackPage,
});

const TABS = ["overview", "compose"] as const;
type Tab = (typeof TABS)[number];

export function ComposeStackPage() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const { add } = useToast();
  const { data: stack, isLoading } = useComposeStack(id);
  const up = useComposeUp();
  const down = useComposeDown();
  const del = useDeleteCompose();
  const [tab, setTab] = useState<Tab>("overview");
  const [confirmDelete, setConfirmDelete] = useState(false);

  const services = useMemo(() => {
    const m = stack?.compose?.match(/^  (\w[\w-]*):\s*$/gm) ?? [];
    return m.map((s) => s.replace(/^\s+|\s*:\s*$/g, ""));
  }, [stack]);

  if (isLoading) return <div className="space-y-lg"><Skeleton variant="card" className="min-h-48" /></div>;
  if (!stack) {
    return (
      <Card><h1 className="text-headline-lg text-foreground">Stack not found</h1><p className="mt-2 text-body-md text-muted-foreground">This compose stack does not exist.</p></Card>
    );
  }

  const running = stack.status === "running";

  const handleDelete = async () => {
    try {
      await del.mutateAsync(id);
      add({ title: "Stack deleted", tone: "success" });
      navigate({ to: "/marketplace" });
    } catch (e) {
      add({ title: "Delete failed", description: e instanceof Error ? e.message : "Unable to delete stack.", tone: "error" });
    }
  };

  return (
    <div className="space-y-lg">
      <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between"><div><h1 className="text-headline-lg text-foreground">{stack.name}</h1><p className="mt-1 text-body-md text-muted-foreground">Docker Compose stack installed from the marketplace or created manually.</p></div><div className="flex flex-wrap items-center gap-2">
            <Badge tone={running ? "success" : "neutral"} dot>{running ? "Running" : "Stopped"}</Badge>
            {running ? (
              <Button
                variant="danger"
                icon={Stop as never}
                onClick={() =>
                  down.mutate(id, {
                    onSuccess: () => add({ title: "Stack stopped", tone: "success" }),
                    onError: (e) => add({ title: "Stop failed", description: e instanceof Error ? e.message : "Unable to stop stack.", tone: "error" }),
                  })
                }
              >
                Stop
              </Button>
            ) : (
              <Button
                icon={Play as never}
                onClick={() =>
                  up.mutate(id, {
                    onSuccess: () => add({ title: "Stack starting", tone: "success" }),
                    onError: (e) => add({ title: "Start failed", description: e instanceof Error ? e.message : "Unable to start stack.", tone: "error" }),
                  })
                }
              >
                Start
              </Button>
            )}
            <Button variant="ghost" icon={Trash as never} onClick={() => setConfirmDelete(true)}>
              Delete
            </Button>
          </div></div>

      <div className="flex items-center gap-sm border-b border-outline-variant mb-lg">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-md py-2.5 font-label-caps text-label-caps uppercase border-b-2 -mb-px transition-colors ${
              tab === t ? "border-primary text-primary" : "border-transparent text-on-surface-variant hover:text-on-surface"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "overview" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-md">
          <Card className="p-md">
            <FileCode size={24} className="text-primary" />
            <p className="font-headline-md font-bold text-on-surface mt-sm">{services.length}</p>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">Services</p>
            <div className="mt-md flex flex-wrap gap-sm">
              {services.map((s) => (
                <span key={s} className="px-2 py-0.5 rounded bg-surface-container-high border border-outline-variant font-code-md text-code-md text-on-surface">
                  {s}
                </span>
              ))}
            </div>
          </Card>
          <Card className="p-md">
            <FileCode size={24} className="text-primary" />
            <p className="font-headline-md font-bold text-on-surface mt-sm">{(stack.compose || "").split("\n").length}</p>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">Lines of compose</p>
          </Card>
          <Card className="p-md">
            <FileCode size={24} className="text-primary" />
            <p className="font-headline-md font-bold text-on-surface mt-sm">{new Date(stack.created_at).toLocaleDateString()}</p>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">Created</p>
          </Card>
        </div>
      )}

      {tab === "compose" && (
        <Card className="p-0 overflow-hidden">
          <div className="flex items-center justify-between px-md py-2 border-b border-outline-variant bg-surface-container-low/50">
            <span className="font-code-md text-[12px] text-on-surface">docker-compose.yml</span>
            <button
              onClick={() => {
                const blob = new Blob([stack.compose], { type: "application/x-yaml" });
                const url = URL.createObjectURL(blob);
                const a = document.createElement("a");
                a.href = url;
                a.download = "docker-compose.yml";
                a.click();
                URL.revokeObjectURL(url);
              }}
              className="px-2 py-1 rounded font-code-md text-[11px] text-on-surface-variant hover:text-on-surface transition-colors"
            >
              Download
            </button>
          </div>
          <div className="h-[60vh]">
            <Editor
              language="yaml"
              value={stack.compose}
              theme="vs-dark"
              options={{ readOnly: true, minimap: { enabled: false }, fontSize: 12.5, fontFamily: 'monospace', lineNumbers: "on", folding: true, automaticLayout: true }}
            />
          </div>
        </Card>
      )}

      <AlertDialog
        trigger={<span />}
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        onConfirm={handleDelete}
        title="Delete stack"
        description={`Remove ${stack.name} and its containers? Volumes will be deleted. This cannot be undone.`}
        confirmLabel="Delete"
      />
    </div>
  );
}
