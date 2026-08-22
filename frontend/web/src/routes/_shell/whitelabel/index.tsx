import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { createFileRoute } from "@tanstack/react-router";
import { useBranding, useSaveBranding } from "../../../hooks";
import { Badge, Button, Card, Field, Input, Skeleton, useToast } from "@aether/design-system";
import { CheckCircle, Cloud, FloppyDisk } from "@phosphor-icons/react";

const brandSchema = z.object({
  name: z.string().min(1, "Name is required"),
  logo_url: z.string().url("Invalid URL").or(z.literal("")),
  primary_color: z.string().regex(/^#[0-9a-fA-F]{6}$/, "Invalid hex"),
  accent_color: z.string().regex(/^#[0-9a-fA-F]{6}$/, "Invalid hex"),
  dark_mode: z.boolean(),
});

export function Whitelabel() {
  const { data: branding, isLoading } = useBranding();
  const save = useSaveBranding();
  const { add } = useToast();
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<z.input<typeof brandSchema>, any, z.output<typeof brandSchema>>({
    resolver: zodResolver(brandSchema),
    defaultValues: { name: "", logo_url: "", primary_color: "#568dff", accent_color: "#568dff", dark_mode: true },
  });

  const name = watch("name") || branding?.name || "Aether";
  const color = watch("primary_color") || branding?.primary_color || "#568dff";

  if (isLoading) return <div className="space-y-md" aria-label="Loading branding settings"><Skeleton variant="card" /><Skeleton variant="card" /></div>;

  const submit = async (values: z.infer<typeof brandSchema>) => {
    try {
      await save.mutateAsync(values);
      add({ title: "Branding saved", description: "Branding was saved and applied.", tone: "success" });
    } catch (err) {
      add({ title: "Save failed", description: err instanceof Error ? err.message : "Unable to save branding.", tone: "error" });
    }
  };

  return (
    <div className="space-y-lg max-w-[1080px]">
      <div><h1 className="text-headline-lg text-foreground">Whitelabeling</h1><p className="mt-1 text-body-md text-muted-foreground">Configure platform branding for your tenant. The primary color is applied to the shell.</p></div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-lg">
        <div className="xl:col-span-2 space-y-lg">
          <Card>
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
              Brand Assets
            </h2>
            <form
              onSubmit={handleSubmit(submit)}
              className="space-y-lg"
              noValidate
              defaultValue={undefined}
            >
              <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
                <Field label="Platform Name" error={errors.name?.message}>
                  <Input placeholder="Aether" defaultValue={branding?.name} {...register("name")} />
                </Field>
                <Field label="Logo URL" description={errors.logo_url?.message || "Leave empty for the default logo."} error={errors.logo_url?.message}>
                  <Input placeholder="https://cdn.example.com/logo.svg" defaultValue={branding?.logo_url} {...register("logo_url")} />
                </Field>
                <Field label="Primary Color" error={errors.primary_color?.message}>
                  <div className="flex items-center gap-sm">
                    <input
                      type="color"
                      defaultValue={branding?.primary_color || "#568dff"}
                      {...register("primary_color")}
                      className="w-10 h-10 rounded-DEFAULT bg-surface border border-outline-variant cursor-pointer"
                    />
                    <Input placeholder="#568dff" defaultValue={branding?.primary_color} {...register("primary_color")} />
                  </div>
                </Field>
                <Field label="Accent Color" error={errors.accent_color?.message}>
                  <Input placeholder="#568dff" defaultValue={branding?.accent_color} {...register("accent_color")} />
                </Field>
                <label className="flex items-center gap-sm cursor-pointer select-none md:col-span-2">
                  <input type="checkbox" defaultChecked={branding?.dark_mode} className="w-4 h-4 rounded-sm bg-surface border-outline-variant text-primary" {...register("dark_mode")} />
                  <span className="font-body-md text-body-md text-on-surface">Dark mode</span>
                </label>
              </div>
              <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
                <Button type="submit" disabled={isSubmitting}>
                  <FloppyDisk size={16} />
                  Save branding
                </Button>
              </div>
            </form>
          </Card>
        </div>

        <div className="space-y-lg">
          <Card className="flex flex-col items-center text-center">
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
              Preview
            </p>
            <div
              className="w-12 h-12 rounded-lg flex items-center justify-center mb-md border border-outline-variant"
              style={{ backgroundColor: color }}
            >
              <Cloud size={22} style={{ color: "#002d6f" }} />
            </div>
            <p className="font-headline-sm text-headline-sm text-on-surface mb-xs">{name}</p>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-md">
              PaaS Platform Control Plane
            </p>
            <div className="w-full">
              <div className="rounded-lg border border-outline-variant overflow-hidden">
                <div className="h-2" style={{ backgroundColor: color }} />
                <div className="p-sm space-y-xs">
                  <div className="h-2 w-3/4 rounded bg-surface-container-high" />
                  <div className="h-2 w-1/2 rounded bg-surface-container-high" />
                </div>
              </div>
            </div>
          </Card>

          <Card>
            <h2 className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-md">
              Tenant status
            </h2>
            <div className="space-y-sm">
              <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
                <span className="font-body-sm text-body-sm text-on-surface">White-label</span>
                <Badge tone={branding?.name ? "success" : "neutral"} dot>{branding?.name ? "Active" : "Disabled"}</Badge>
              </div>
              <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
                <span className="font-body-sm text-body-sm text-on-surface">Primary color</span>
                <span className="font-code-md text-code-md text-on-surface-variant">{branding?.primary_color || "#568dff"}</span>
              </div>
              <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
                <span className="font-body-sm text-body-sm text-on-surface">Last updated</span>
                <span className="font-code-md text-code-md text-on-surface-variant">
                  {branding?.updated_at ? new Date(branding.updated_at).toLocaleString() : "never"}
                </span>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}

export const Route = createFileRoute("/_shell/whitelabel/")({
  component: Whitelabel,
});
