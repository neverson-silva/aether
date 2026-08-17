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
import { Card, StatusPill, Table, Button } from "../../../components/ui";
import { AppPage, AppPageHeader } from "../../../components/ds";

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
    window.alert(`Pull image via CLI: skopeo copy docker://${values.repo}:${values.tag} docker://registry.internal/${values.repo}:${values.tag}`);
    reset();
  });

  const onRefresh = () => images.refetch();

  return (
    <AppPage>
      <AppPageHeader
        title="Image Registry"
        description="Internal OCI registry with push/pull via Skopeo."
        actions={
          <>
            <StatusPill status={enabled ? "running" : "disabled"} />
            <Button variant={enabled ? "ghost" : "primary"} onClick={onToggle}>
              {enabled ? "Disable" : "Enable"}
            </Button>
          </>
        }
      />

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
              <input
                {...register("repo")}
                placeholder="nginx"
                className="w-full bg-surface-container-low border border-outline-variant rounded-md px-sm py-2 font-code-md text-code-md text-on-surface"
              />
              <input
                {...register("tag")}
                placeholder="tag (e.g. alpine)"
                className="w-full bg-surface-container-low border border-outline-variant rounded-md px-sm py-2 font-code-md text-code-md text-on-surface"
              />
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
        <Table headers={["Repository", "Tags", "Size", ""]}>
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
        </Table>
      </div>
    </AppPage>
  );
}

export const Route = createFileRoute("/_shell/registry/")({
  component: Registry,
});
