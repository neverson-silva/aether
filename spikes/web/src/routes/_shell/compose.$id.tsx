import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import Editor from "@monaco-editor/react";
import { AppPage, AppPageHeader, AppCard, AppLoading, AppEmptyState, AppStatusBadge } from "../../components/ds";
import { Button, useToast } from "../../components/ui";
import { useComposeStack, useComposeUp, useComposeDown, useDeleteCompose } from "../../api/hooks";

export const Route = createFileRoute("/_shell/compose/$id")({
  component: ComposeStackPage,
});

const TABS = ["overview", "compose"] as const;
type Tab = (typeof TABS)[number];

export function ComposeStackPage() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const { toast } = useToast();
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

  if (isLoading) return <AppLoading label="Loading compose stack..." />;
  if (!stack) {
    return (
      <AppPage>
        <AppEmptyState icon="deployed_code" title="Stack not found" description="This compose stack does not exist." />
      </AppPage>
    );
  }

  const running = stack.status === "running";

  const handleDelete = async () => {
    try {
      await del.mutateAsync(id);
      toast("Stack deleted");
      navigate({ to: "/marketplace" });
    } catch (e) {
      toast(e instanceof Error ? e.message : "failed", "error");
    }
  };

  return (
    <AppPage>
      <AppPageHeader
        title={stack.name}
        description="Docker Compose stack installed from the marketplace or created manually."
        actions={
          <>
            <AppStatusBadge status={running ? "running" : "stopped"} pulse={running} />
            {running ? (
              <Button variant="danger" leftIcon="stop_circle" onClick={() => down.mutate(id)}>
                Stop
              </Button>
            ) : (
              <Button leftIcon="play_arrow" onClick={() => up.mutate(id)}>
                Start
              </Button>
            )}
            <Button variant="ghost" leftIcon="delete" onClick={() => setConfirmDelete(true)}>
              Delete
            </Button>
          </>
        }
      />

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
          <AppCard className="p-md">
            <span className="material-symbols-outlined text-primary text-2xl">layers</span>
            <p className="font-headline-md font-bold text-on-surface mt-sm">{services.length}</p>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">Services</p>
            <div className="mt-md flex flex-wrap gap-sm">
              {services.map((s) => (
                <span key={s} className="px-2 py-0.5 rounded bg-surface-container-high border border-outline-variant font-code-md text-code-md text-on-surface">
                  {s}
                </span>
              ))}
            </div>
          </AppCard>
          <AppCard className="p-md">
            <span className="material-symbols-outlined text-primary text-2xl">deployed_code</span>
            <p className="font-headline-md font-bold text-on-surface mt-sm">{(stack.compose || "").split("\n").length}</p>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">Lines of compose</p>
          </AppCard>
          <AppCard className="p-md">
            <span className="material-symbols-outlined text-primary text-2xl">schedule</span>
            <p className="font-headline-md font-bold text-on-surface mt-sm">{new Date(stack.created_at).toLocaleDateString()}</p>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase">Created</p>
          </AppCard>
        </div>
      )}

      {tab === "compose" && (
        <AppCard className="p-0 overflow-hidden">
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
        </AppCard>
      )}

      {confirmDelete && (
        <div className="fixed inset-0 z-[80] flex items-center justify-center bg-black/60 p-4" onClick={() => setConfirmDelete(false)}>
          <div onClick={(e) => e.stopPropagation()} className="bg-surface-container-low border border-outline-variant rounded-xl w-full max-w-md p-lg shadow-2xl">
            <h2 className="font-headline-sm font-bold text-on-surface mb-xs">Delete stack</h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-lg">Remove {stack.name} and its containers? Volumes will be deleted. This cannot be undone.</p>
            <div className="flex justify-end gap-sm">
              <Button variant="ghost" onClick={() => setConfirmDelete(false)}>Cancel</Button>
              <Button variant="danger" onClick={handleDelete}>Delete</Button>
            </div>
          </div>
        </div>
      )}
    </AppPage>
  );
}
