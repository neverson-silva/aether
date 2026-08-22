import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { CaretRight, Pulse, SpinnerGap } from "@phosphor-icons/react";
import { useCreateApp, useCreateCompose, useProjects, useTrendingTemplates } from "../hooks";
import { getServer } from "../api/client";
import { Button, Dialog, Field, Input, NativeSelect, useToast } from "@aether/design-system";
import { ComposeEditor } from "./ComposeEditor";

const createSchema = z
  .object({
    project_id: z.string().min(1, "Project is required"),
    environment_id: z.string().optional(),
    name: z.string().min(1, "Name is required").max(64),
    source_type: z.enum(["image", "git", "compose", "template"]),
    image: z.string().optional(),
    git_url: z.string().optional(),
    git_branch: z.string().default("main"),
    dockerfile: z.string().default("Dockerfile"),
    build_type: z.enum(["dockerfile", "buildpacks"]).default("dockerfile"),
    port: z.coerce.number().int().min(1).max(65535).default(80),
    mem_mb: z.coerce.number().int().min(0).default(0),
    health_enabled: z.boolean().default(false),
    health_path: z.string().default("/"),
  })
  .refine((v) => (v.source_type === "image" ? !!v.image : !!v.git_url), {
    message: "Fill in the image or repository according to the source",
    path: ["image"],
  });

type CreateForm = z.infer<typeof createSchema>;

const POPULAR_IMAGES = ["nginx:alpine", "postgres:16", "redis:7", "mysql:8.4", "mongo:7", "node:22-alpine", "python:3.12-slim", "ghost:5"];

function StatusChip({ text }: { text: string }) {
  return (
    <span className="px-2 py-0.5 rounded border border-primary/40 font-code-md text-code-md text-primary">{text}</span>
  );
}

export function CreateServiceModal({
  open,
  onClose,
  fixedProjectId,
}: {
  open: boolean;
  onClose: () => void;
  fixedProjectId?: string;
}) {
  const { data: projects } = useProjects();
  const { data: trendingTemplates } = useTrendingTemplates();
  const [imageSuggest, setImageSuggest] = useState(false);
  const createApp = useCreateApp();
  const createCompose = useCreateCompose();
  const { add } = useToast();
  const [composeYAML, setComposeYAML] = useState(`services:
  app:
    image: nginx:alpine
    ports:
      - "80:80"`);
  const [detecting, setDetecting] = useState(false);
  const [detected, setDetected] = useState<{ framework: string; build_method: string; start_command: string; build_command: string; port: number; detected: boolean } | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<z.input<typeof createSchema>, any, z.output<typeof createSchema>>({
    resolver: zodResolver(createSchema),
    defaultValues: {
      project_id: fixedProjectId ?? "",
      source_type: "image",
      git_branch: "main",
      dockerfile: "Dockerfile",
      build_type: "dockerfile",
      port: 80,
      mem_mb: 0,
      health_enabled: false,
      health_path: "/",
    },
  });

  const source = watch("source_type");
  const gitURL = watch("git_url");

  const detectFramework = async () => {
    if (!gitURL) {
      add({ title: "Enter a repository URL first", tone: "info" });
      return;
    }
    setDetecting(true);
    setDetected(null);
    try {
      const resp = await fetch(`${getServer()}/api/v1/detect`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ git_url: gitURL, git_branch: watch("git_branch") }),
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error || "detection failed");
      setDetected(data);
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Detection failed", tone: "error" });
    } finally {
      setDetecting(false);
    }
  };

  const submit = async (values: CreateForm) => {
    try {
      if (values.source_type === "compose") {
        await createCompose.mutateAsync({ project_id: values.project_id, name: values.name, content: composeYAML });
        onClose();
        reset();
        return;
      }
      await createApp.mutateAsync({
        projectID: values.project_id,
        payload: {
          name: values.name,
          environment_id: values.environment_id ?? "",
          source_type: values.source_type === "template" ? "image" : values.source_type,
          image: values.image || "",
          git_url: values.git_url || "",
          git_branch: values.git_branch,
          dockerfile: values.dockerfile,
          build_type: values.build_type,
          port: values.port,
          resources: { cpus: "", mem_mb: values.mem_mb },
          health_check: {
            enabled: values.health_enabled,
            path: values.health_path,
            interval_ms: 5000,
            timeout_ms: 2000,
            retries: 3,
          },
        },
      });
      onClose();
      reset();
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Failed to create service", tone: "error" });
    }
  };

  return (
    <Dialog trigger={<span />} open={open} onOpenChange={(value) => { if (!value) onClose(); }} title="Create service">
      <form onSubmit={handleSubmit(submit)} className="space-y-lg" noValidate>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
          <Field label="Project" error={errors.project_id?.message}>
            <NativeSelect {...register("project_id")} disabled={!!fixedProjectId} options={[{ label: "Select...", value: "" }, ...(projects ?? []).map((p) => ({ label: p.name, value: p.id }))]} />
          </Field>
          <Field label="Name" error={errors.name?.message}>
            <Input placeholder="ex: api-gateway" {...register("name")} />
          </Field>
          <Field label="Source">
            <NativeSelect {...register("source_type")} options={[{ label: "OCI Image", value: "image" }, { label: "Git Repository", value: "git" }, { label: "Docker Compose", value: "compose" }, { label: "Template", value: "template" }]} />
          </Field>
          <Field label="Container port" error={errors.port?.message}>
            <Input type="number" {...register("port")} />
          </Field>
        </div>

        {source === "template" ? (
          <div>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase mb-sm">Popular templates</p>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-sm max-h-[240px] overflow-y-auto sidebar-scroll">
              {(trendingTemplates ?? []).slice(0, 18).map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => {
                    setValue("image", t.name.toLowerCase().includes("postgres") ? "postgres:16" : t.name.toLowerCase().includes("redis") ? "redis:7" : t.name.toLowerCase().includes("mysql") ? "mysql:8.4" : t.name.toLowerCase().includes("mongo") ? "mongo:7" : t.name.toLowerCase().includes("nginx") ? "nginx:alpine" : "ghcr.io/" + t.name.toLowerCase().replace(/\s+/g, "-"), { shouldValidate: true });
                  }}
                  className="flex items-center gap-sm p-sm rounded border border-outline-variant hover:border-primary/50 transition-colors text-left bg-surface-container-low"
                >
                  <span className="text-[22px]">{t.icon}</span>
                  <span className="min-w-0">
                    <span className="block font-body-sm text-body-sm text-on-surface truncate">{t.name}</span>
                    <span className="block font-code-md text-code-md text-on-surface-variant/60 truncate">{t.description}</span>
                  </span>
                </button>
              ))}
            </div>
          </div>
        ) : source === "compose" ? (
          <ComposeEditor value={composeYAML} onChange={setComposeYAML} />
        ) : source === "image" ? (
          <div className="relative">
            <Field label="Image" error={errors.image?.message} description={!errors.image?.message ? "ex: nginx:alpine" : undefined}>
              <Input placeholder="registry/repo:tag" {...register("image")} onFocus={() => setImageSuggest(true)} onBlur={() => setTimeout(() => setImageSuggest(false), 150)} />
            </Field>
            {imageSuggest && (
              <div className="absolute z-10 w-full mt-1 bg-surface-modal border border-outline-variant rounded-lg shadow-lg py-1">
                {POPULAR_IMAGES.map((img) => (
                  <button
                    key={img}
                    type="button"
                    onMouseDown={(e) => {
                      e.preventDefault();
                      setValue("image", img, { shouldValidate: true });
                      setImageSuggest(false);
                    }}
                    className="w-full text-left px-sm py-1.5 font-code-md text-code-md text-on-surface hover:bg-surface-container-high"
                  >
                    {img}
                  </button>
                ))}
              </div>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-lg">
            <div className="md:col-span-2">
              <Field label="Repository URL" error={errors.git_url?.message}>
                <Input placeholder="git@github.com:org/repo.git" {...register("git_url")} />
              </Field>
            </div>
            <div className="flex items-end">
              <Button type="button" variant="ghost" disabled={detecting || !gitURL} onClick={detectFramework}>
                {detecting ? <SpinnerGap size={16} className="animate-spin" aria-hidden="true" /> : <Pulse size={16} aria-hidden="true" />}
                {detecting ? "Detecting..." : "Detect framework"}
              </Button>
            </div>
            <Field label="Branch" error={errors.git_branch?.message}>
              <Input placeholder="main" {...register("git_branch")} />
            </Field>
            <div className="md:col-span-2">
              <Field label="Dockerfile" error={errors.dockerfile?.message}>
                <Input placeholder="Dockerfile" {...register("dockerfile")} />
              </Field>
            </div>
            <Field label="Build method">
              <NativeSelect {...register("build_type")} options={[{ label: "Dockerfile", value: "dockerfile" }, { label: "SmartBuild (CNB)", value: "buildpacks" }]} />
            </Field>
          </div>
        )}

        {detected && (
          <div className="bg-surface-container-low border border-primary/30 rounded-lg p-md space-y-sm">
            <div className="flex items-center justify-between">
              <p className="font-label-caps text-label-caps text-primary uppercase">Framework detected</p>
              <StatusChip text={detected.framework} />
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-md">
              <div>
                <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase">Build</p>
                <p className="font-code-md text-code-md text-on-surface">{detected.build_method || "buildpacks"}</p>
              </div>
              <div>
                <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase">Start</p>
                <p className="font-code-md text-code-md text-on-surface">{detected.start_command || "—"}</p>
              </div>
              <div>
                <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase">Port</p>
                <p className="font-code-md text-code-md text-on-surface">{detected.port}</p>
              </div>
              <div>
                <p className="font-label-caps text-label-caps text-on-surface-variant/60 uppercase">Build cmd</p>
                <p className="font-code-md text-code-md text-on-surface">{detected.build_command || "—"}</p>
              </div>
            </div>
          </div>
        )}
        <details className="group border border-outline-variant rounded-lg">
          <summary className="flex items-center gap-sm px-md py-2.5 font-label-caps text-label-caps text-on-surface-variant uppercase cursor-pointer select-none">
            <CaretRight size={16} className="transition-transform group-open:rotate-90" aria-hidden="true" />
            Advanced settings
          </summary>
          <div className="px-md pb-md space-y-lg border-t border-outline-variant/60 pt-lg">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
          <Field label="Memory (MiB)" error={errors.mem_mb?.message} description={!errors.mem_mb?.message ? "0 = unlimited" : undefined}>
            <Input type="number" {...register("mem_mb")} />
          </Field>
          <Field label="Health check path" error={errors.health_path?.message}>
            <Input placeholder="/" {...register("health_path")} />
          </Field>
        </div>

        <label className="flex items-center gap-sm cursor-pointer select-none">
          <input type="checkbox" className="w-4 h-4 rounded-sm bg-surface border-outline-variant text-primary" {...register("health_enabled")} />
          <span className="font-body-md text-body-md text-on-surface">Enable health check (required for auto-rollback)</span>
        </label>
          </div>
        </details>

        <div className="flex justify-end gap-md border-t border-outline-variant pt-lg">
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            Create Service
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
