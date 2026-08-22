import { createFileRoute } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  useRegistrySettings,
  useRegistryImages,
  useToggleRegistry,
  useMirrors,
  useCreateMirror,
  useRunMirror,
  useDeleteMirror,
} from "../../../hooks";
import { Badge, Button, Card, Input, useToast } from "@aether/design-system";

const schema = z.object({
  repo: z.string().min(1),
  tag: z.string().min(1),
});
type Form = z.input<typeof schema>;
type Filled = z.output<typeof schema>;

function fmtBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MiB`;
}

function Registry() {
  const settings = useRegistrySettings();
  const images = useRegistryImages();
  const toggle = useToggleRegistry();
  const mirrors = useMirrors();
  const createMirror = useCreateMirror();
  const runMirror = useRunMirror();
  const deleteMirror = useDeleteMirror();
  const { add } = useToast();
  const { register, handleSubmit, reset } = useForm<Form, unknown, Filled>({
    resolver: zodResolver(schema),
    defaultValues: { repo: "", tag: "" },
  });

  const enabled = settings.data?.enabled ?? false;

  const onToggle = () => {
    toggle.mutate(!enabled, {
      onSuccess: () => {
        if (!enabled) images.refetch();
      },
    });
  };

  const onPull = handleSubmit((values) => {
    add({ title: "Push command ready", description: `skopeo copy docker://${values.repo}:${values.tag} docker://registry.internal/${values.repo}:${values.tag}`, tone: "info", persistent: true });
    reset();
  });

  const onRefresh = () => images.refetch();

  return (
    <main className="mx-auto flex w-full max-w-screen-2xl flex-col gap-8 p-6 lg:p-8">
      <header className="flex flex-wrap items-end justify-between gap-4"><div><p className="text-label-caps text-primary">Infrastructure</p><h1 className="text-headline-sm font-semibold text-foreground">Image registry</h1><p className="mt-1 text-body-md text-muted-foreground">Internal OCI registry with push and pull via Skopeo.</p></div><div className="flex items-center gap-3"><Badge tone={enabled ? "success" : "neutral"}>{enabled ? "Enabled" : "Disabled"}</Badge><Button variant={enabled ? "ghost" : "primary"} onClick={onToggle}>{enabled ? "Disable" : "Enable"}</Button></div></header>

      {enabled && (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-lg">
          <Card>
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
              Registry endpoint
            </h2>
            <p className="font-code-md text-code-md text-primary">
              {settings.data?.host}:{settings.data?.port}
            </p>
            <p className="font-body-sm text-body-sm text-on-surface-variant mt-sm">
              Images are pushed automatically after each successful deploy.
            </p>
          </Card>
          <Card>
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
              Push from CLI
            </h2>
            <form onSubmit={onPull} className="space-y-sm">
              <Input {...register("repo")} placeholder="nginx" />
              <Input {...register("tag")} placeholder="tag (e.g. alpine)" />
              <Button type="submit" className="w-full">
                Show push command
              </Button>
            </form>
          </Card>
          <Card>
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
              Storage
            </h2>
            <p className="font-body-sm text-body-sm text-on-surface-variant">
              Layer deduplication enabled. Delete images via the table below.
            </p>
          </Card>
        </div>
      )}

      <div className="bg-surface-container-low border border-outline-variant rounded-lg">
        <div className="flex items-center justify-between px-sm py-sm">
          <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase">
            Images
          </h2>
          <Button variant="ghost" onClick={onRefresh} disabled={!enabled}>
            Refresh
          </Button>
        </div>
        <table className="w-full text-left"><thead><tr className="border-b border-border text-label-caps text-muted-foreground"><th className="px-3 py-3">Repository</th><th className="px-3 py-3">Tags</th><th className="px-3 py-3">Size</th><th className="px-3 py-3"></th></tr></thead><tbody>
          {(images.data ?? []).length === 0 && (
            <tr>
              <td colSpan={4} className="px-sm py-lg font-body-sm text-body-sm text-on-surface-variant text-center">
                {enabled ? "No images yet. Deploy an app to push its image here." : "Enable the registry to start collecting images."}
              </td>
            </tr>
          )}
          {(images.data ?? []).map((img) => (
            <tr key={img.repo} className="hover:bg-surface-container-high transition-colors">
              <td className="px-sm py-2 font-code-md text-code-md text-primary">{img.repo}</td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">
                {img.tags?.join(", ") || ""}
              </td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant">
                {fmtBytes(img.size)}
              </td>
              <td className="px-sm py-2 font-code-md text-code-md text-on-surface-variant/60">-</td>
            </tr>
          ))}
        </tbody></table>
      </div>
    </main>
  );
}

export const Route = createFileRoute("/_shell/registry/")({
  component: Registry,
});
