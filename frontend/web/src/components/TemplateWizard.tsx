import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  ArrowRight,
  CaretDown,
  Check,
  List,
  MagnifyingGlass,
  NotePencil,
  ShoppingBag,
} from "@phosphor-icons/react";
import {
  Button,
  Input,
  Modal,
  Select,
  VariableEditor,
  useToast,
} from "@aether/design-system";
import {
  useCreateApp,
  useInstallTemplate,
  useProjects,
  useTemplateCategories,
  useTemplatesFiltered,
} from "../hooks";
import type { TemplateItem } from "../hooks";
import { TemplateIcon } from "./TemplateIcon";

interface ParsedDef {
  services: {
    name: string;
    image: string;
    port: number;
    versions?: string[];
    env?: Record<string, string>;
  }[];
}

function parseDef(t: TemplateItem): ParsedDef {
  try {
    const parsed: unknown = JSON.parse(t.definition || "{}");
    if (!parsed || typeof parsed !== "object") return { services: [] };
    const services = (parsed as Record<string, unknown>).services;
    return {
      services: Array.isArray(services)
        ? (services as ParsedDef["services"])
        : [],
    };
  } catch {
    return { services: [] };
  }
}

function genPassword(len = 24): string {
  const chars =
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
  let out = "";
  const arr = new Uint32Array(len);
  crypto.getRandomValues(arr);
  for (let i = 0; i < len; i++) out += chars[arr[i] % chars.length];
  return out;
}

function resolveTemplateValue(
  value: string,
  variables: Record<string, string>,
  cache: Map<string, string>,
  seed: string,
  stack = new Set<string>(),
): string {
  return value.replace(/\$\{([^}]+)\}/g, (_, token: string) => {
    const parts = token.trim().split(":");
    const name = parts[0].toLowerCase();
    if (token.trim() in variables && !stack.has(token.trim())) {
      const key = token.trim();
      const cached = cache.get(key);
      if (cached !== undefined) return cached;
      const next = new Set(stack);
      next.add(key);
      const resolved = resolveTemplateValue(
        variables[key],
        variables,
        cache,
        seed,
        next,
      );
      cache.set(key, resolved);
      return resolved;
    }
    const length = Number(parts[1]) > 0 ? Number(parts[1]) : 32;
    if (name === "domain")
      return `${seed.toLowerCase().replace(/[^a-z0-9-]+/g, "-")}.localhost`;
    if (name === "password") return genPassword(length);
    if (name === "hash")
      return genPassword(length)
        .replace(/[^a-z0-9]/gi, "a")
        .toLowerCase();
    if (name === "base64")
      return btoa(
        String.fromCharCode(...crypto.getRandomValues(new Uint8Array(length))),
      );
    if (name === "uuid") return crypto.randomUUID();
    if (name === "randomport")
      return String(1024 + Math.floor(Math.random() * 64512));
    if (name === "timestamp") return String(Date.now());
    if (name === "timestamps") return String(Math.round(Date.now() / 1000));
    if (name === "timestampms") return String(Date.now());
    if (name === "email")
      return `user-${genPassword(8).toLowerCase()}@example.com`;
    if (name === "username") return `user${genPassword(8).toLowerCase()}`;
    if (name === "jwt")
      return genPassword(
        parts.length === 2 && Number.isFinite(length) ? length : 32,
      );
    return `\${${token}}`;
  });
}

function resolveEnvironmentValue(
  name: string,
  value: string,
  variables: Record<string, string>,
  cache: Map<string, string>,
): string {
  if (value) return resolveTemplateValue(value, variables, cache, name);
  return /password|passwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|credential/i.test(
    name,
  )
    ? genPassword()
    : "";
}

function templateEnvironment(
  t: TemplateItem,
): { name: string; value: string }[] {
  const variables = Object.fromEntries(
    (t.variables ?? []).map((entry) => [entry.name, entry.value]),
  );
  for (const entry of t.environment ?? []) {
    if (!(entry.name in variables)) variables[entry.name] = entry.value;
  }
  const cache = new Map<string, string>();
  if (t.environment?.length)
    return t.environment.map((entry) => ({
      name: entry.name,
      value: resolveEnvironmentValue(entry.name, entry.value, variables, cache),
    }));
  const svc = parseDef(t).services[0];
  return Object.entries(svc?.env ?? {}).map(([name, value]) => ({
    name,
    value: resolveEnvironmentValue(name, value, variables, cache),
  }));
}

function iconTone(t: TemplateItem): string {
  const cat = t.category.toLowerCase();
  if (cat === "monitoring" || cat === "logging" || cat === "analytics")
    return "bg-secondary/10 text-secondary";
  if (cat === "ai" || cat === "automation")
    return "bg-tertiary-container/20 text-tertiary-container";
  if (cat === "security" || cat === "identity")
    return "bg-surface-container-high text-on-surface-variant";
  return "bg-primary/10 text-primary";
}

const CPU_SEGS = [".25", ".5", "1", "2", "4", "8"];
const RAM_SEGS = [".25", ".5", "1", "2", "4", "8", "∞"];
const VOL_SEGS = ["1G", "5G", "10G", "50G", "100G", "∞"];

function fmtCpu(c: string): string {
  return c.startsWith(".") ? "0" + c : c;
}

const segmentButtonClass =
  "inline-flex min-h-8 items-center justify-center rounded-lg border border-outline-variant bg-surface-container-low px-sm py-xs font-code-md text-code-md text-on-surface-variant transition-[background-color,border-color,color,box-shadow,transform] duration-150 hover:border-primary/60 hover:text-on-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.98]";

export function TemplateWizard({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const { data: projects } = useProjects();
  const { data: categories } = useTemplateCategories();
  const createApp = useCreateApp();
  const installTemplate = useInstallTemplate();
  const { add } = useToast();
  const navigate = useNavigate();

  const [step, setStep] = useState<1 | 2>(1);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("");
  const [selected, setSelected] = useState<TemplateItem | null>(null);
  const [projectId, setProjectId] = useState("");
  const [name, setName] = useState("");
  const [version, setVersion] = useState("");
  const [envs, setEnvs] = useState<{ name: string; value: string }[]>([]);
  const [initialEnvs, setInitialEnvs] = useState<
    { name: string; value: string }[]
  >([]);
  const [cpu, setCpu] = useState(".5");
  const [ram, setRam] = useState(".5");
  const [vol, setVol] = useState("∞");
  const [creating, setCreating] = useState(false);
  const creatingRef = useRef(false);

  useEffect(() => {
    if (open) {
      setStep(1);
      setSelected(null);
      setQuery("");
      setCategory("");
    }
  }, [open]);

  const { data: templates } = useTemplatesFiltered({
    category: category || undefined,
    q: query || undefined,
  });

  const pick = (t: TemplateItem) => {
    const def = parseDef(t);
    const svc = def.services[0];
    setSelected(t);
    setVersion(svc?.versions?.[0] ?? "");
    setStep(2);
    setName(
      t.name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, "") +
        "-" +
        Math.floor(Math.random() * 900 + 100),
    );
    const defaults = templateEnvironment(t);
    setEnvs(defaults);
    setInitialEnvs(defaults);
  };

  const envCount = (t: TemplateItem): number => {
    if (t.environment?.length) return t.environment.length;
    return Object.keys(parseDef(t).services[0]?.env ?? {}).length;
  };

  const create = async () => {
    if (creatingRef.current) return;
    if (!selected || !projectId || !name.trim()) {
      add({ title: "Select a project and set a name", tone: "error" });
      return;
    }
    creatingRef.current = true;
    setCreating(true);
    try {
      if (selected.compose_yaml || selected.remote_id) {
        const defaults = new Map(
          initialEnvs.map((entry) => [entry.name, entry.value]),
        );
        const overrides = Object.fromEntries(
          envs
            .filter(
              (entry) =>
                entry.name.trim() && defaults.get(entry.name) !== entry.value,
            )
            .map((entry) => [entry.name.trim(), entry.value]),
        );
        await installTemplate.mutateAsync({
          id: selected.id,
          project_id: projectId,
          name,
          overrides,
        });
        add({
          title: "Deploy it manually from the services page",
          tone: "info",
        });
        onClose();
        navigate({ to: "/apps" } as never);
        return;
      }
      const svc = parseDef(selected).services[0];
      const imageRef = version
        ? `${svc?.image}:${version}`
        : (svc?.image ?? selected.name);
      const memMB =
        ram === "∞" ? 0 : Math.round(parseFloat(fmtCpu(ram)) * 1024);
      const storageMB =
        vol === "∞" ? 0 : Math.round(parseFloat(vol.replace("G", "")) * 1024);
      const app = await createApp.mutateAsync({
        projectID: projectId,
        payload: {
          name,
          source_type: "image",
          image: imageRef,
          port: svc?.port ?? 80,
          build_type: "dockerfile",
          resources: {
            cpus: fmtCpu(cpu),
            mem_mb: memMB,
            storage_mb: storageMB,
          },
          health_check: {
            enabled: false,
            path: "/",
            interval_ms: 5000,
            timeout_ms: 2000,
            retries: 3,
          },
          env: envs
            .filter((e) => e.name.trim())
            .map((e) => ({
              name: e.name,
              value: e.value,
              secret: /password|secret|key|token/i.test(e.name),
            })),
        } as never,
      });
      add({ title: "Deploy it manually from the service page", tone: "info" });
      onClose();
      navigate({
        to: "/apps/$appId",
        params: { appId: app.service_id ?? app.id },
      } as never);
    } catch (err) {
      add({
        title: err instanceof Error ? err.message : "Failed to create service",
        tone: "error",
      });
    } finally {
      creatingRef.current = false;
      setCreating(false);
    }
  };

  const svcOfSelected = selected ? parseDef(selected).services[0] : null;
  const isCompose = !!selected?.compose_yaml || !!selected?.remote_id;

  return (
    <Modal
      open={open}
      onOpenChange={(value) => {
        if (!value) onClose();
      }}
      size="wizard"
      showHeader={false}
    >
      <div className="flex h-full min-h-0 flex-col overflow-hidden">
        <header className="flex shrink-0 flex-col gap-sm border-b border-border bg-gradient-to-br from-primary/10 via-surface-card to-secondary/5 px-lg py-lg">
          <div className="flex items-start justify-between">
            <div>
              <p className="font-label-caps text-label-caps uppercase text-primary">
                Template library
              </p>
              <h1 className="mt-xs flex items-center gap-sm font-display-md text-headline-sm tracking-[-0.025em] text-on-surface">
                <ShoppingBag
                  size={22}
                  className="text-tertiary-container"
                  aria-hidden="true"
                />
                {step === 1
                  ? "Templates · Choose one to configure"
                  : `${selected?.name} · Configure`}
              </h1>
              <p className="mt-xs max-w-2xl text-body-md text-on-surface-variant">
                {step === 1
                  ? "One-click templates with default environment variables, ready to configure and deploy."
                  : "Pre-filled with the template's default environment variables — adjust as needed."}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 font-label-caps text-label-caps">
            <div
              className={`flex items-center gap-1.5 ${step === 1 ? "text-primary" : "text-on-surface-variant"}`}
            >
              <div
                className={`flex size-6 items-center justify-center rounded-full border ${step === 1 ? "border-primary bg-primary text-primary-foreground" : "border-status-success bg-status-success text-status-success-foreground"}`}
              >
                {step === 1 ? "1" : "✓"}
              </div>
              <span>Browse</span>
            </div>
            <div className="h-px w-8 bg-border" />
            <div
              className={`flex items-center gap-1.5 ${step === 2 ? "text-primary" : "text-on-surface-variant"}`}
            >
              <div
                className={`flex size-6 items-center justify-center rounded-full border ${step === 2 ? "border-primary bg-primary text-primary-foreground" : "border-border bg-surface-card"}`}
              >
                2
              </div>
              <span>Configure</span>
            </div>
          </div>
        </header>

        <div className="flex shrink-0 flex-col gap-sm border-b border-border bg-surface-container/35 px-lg py-md sm:flex-row sm:items-center">
          <div className="relative min-w-0 flex-1">
            <MagnifyingGlass
              size={16}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant"
              aria-hidden="true"
            />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search templates..."
              className="h-10 w-full rounded-xl border border-border bg-surface-control pl-9 pr-3 text-body-sm text-on-surface outline-none transition-[background-color,border-color,box-shadow] duration-200 hover:bg-surface-container-highest/40 focus:border-primary focus:ring-2 focus:ring-ring/20"
            />
          </div>
          <div className="relative shrink-0">
            <Select
              label=""
              value={category}
              onValueChange={(value) => setCategory(value ?? "")}
              placeholder="All categories"
              options={[
                { label: "All categories", value: "" },
                ...(categories ?? []).map((c) => ({ label: c, value: c })),
              ]}
              className="w-full sm:w-56"
            />
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto p-lg scrollbar-hide">
          {step === 1 ? (
            <>
              <div className="grid grid-cols-1 gap-md md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                {(templates ?? []).map((t) => (
                  <button
                    key={t.id}
                    onClick={() => pick(t)}
                    className="group relative flex h-[180px] cursor-pointer flex-col justify-between rounded-2xl border border-border bg-surface-container-low p-md text-left transition-[border-color,background-color,box-shadow,transform] duration-200 hover:-translate-y-0.5 hover:border-primary/60 hover:bg-surface-container hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {t.featured && (
                      <div className="absolute right-2 top-2 rounded-full bg-tertiary-container/20 px-1.5 py-0.5 font-label-caps text-[9px] text-tertiary-container">
                        FEATURED
                      </div>
                    )}
                    <div className="flex flex-col items-center text-center">
                      <div
                        className={`mb-3 flex size-14 items-center justify-center rounded-2xl ${iconTone(t)}`}
                      >
                        <TemplateIcon template={t} size={30} />
                      </div>
                      <h3 className="font-body-md text-body-md font-semibold text-on-surface truncate w-full">
                        {t.name}
                      </h3>
                      <p className="font-body-sm text-body-sm text-on-surface-variant line-clamp-2 mt-1">
                        {t.description}
                      </p>
                    </div>
                    <div className="flex justify-between items-center mt-4">
                      <div className="flex items-center gap-1 text-on-surface-variant font-label-caps text-label-caps">
                        <List size={12} aria-hidden="true" /> {envCount(t)} env
                        {envCount(t) === 1 ? "" : "s"}
                      </div>
                      <span className="font-label-caps text-label-caps text-primary opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                        Choose <ArrowRight size={12} aria-hidden="true" />
                      </span>
                    </div>
                  </button>
                ))}
              </div>
              {(templates ?? []).length === 0 && (
                <p className="font-body-sm text-body-sm text-on-surface-variant py-md text-center">
                  No templates match "{query}".
                </p>
              )}
            </>
          ) : (
            <div className="p-lg flex flex-col gap-lg">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-md">
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">
                    Project
                  </label>
                  <div className="relative">
                    <Select
                      label=""
                      value={projectId}
                      onValueChange={(value) => setProjectId(value ?? "")}
                      placeholder="Select project..."
                      options={[
                        { label: "Select project...", value: "" },
                        ...(projects ?? []).map((p) => ({
                          label: p.name,
                          value: p.id,
                        })),
                      ]}
                    />
                  </div>
                </div>
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">
                    Name
                  </label>
                  <div className="relative">
                    <Input
                      placeholder="my-service"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                    />
                  </div>
                </div>
              </div>

              {isCompose ? (
                <div className="flex flex-col gap-3">
                  <p className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">
                    Managed stack
                  </p>
                  <div className="rounded-2xl border border-outline-variant bg-surface-container-low p-md">
                    <p className="font-body-sm text-body-sm text-on-surface-variant">
                      This template deploys a full multi-service stack using its
                      official Docker Compose (application + database + cache).
                      Just pick a project and a name — the stack runs as-is.
                      Start it from the services page afterwards.
                    </p>
                  </div>
                </div>
              ) : (
                <>
                  <div className="flex flex-col gap-3">
                    <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">
                      Version
                    </label>
                    <div className="flex flex-wrap items-center gap-2">
                      {(svcOfSelected?.versions ?? []).map((v) => (
                        <button
                          key={v}
                          onClick={() => setVersion(v)}
                          className={`${segmentButtonClass} ${version === v ? "border-primary bg-primary/15 text-primary" : ""}`}
                          type="button"
                        >
                          {v}
                        </button>
                      ))}
                      <span className="font-code-md text-[10px] text-on-surface-variant/70 ml-sm">
                        Image: {svcOfSelected?.image}:{version || "?"}
                      </span>
                    </div>
                    <p className="font-code-md text-[10px] text-on-surface-variant/50">
                      Versions are pinned (never :latest) for reproducible
                      deployments.
                    </p>
                  </div>

                  <div className="flex flex-col gap-3">
                    <p className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">
                      Environment variables (defaults from template)
                    </p>
                    <VariableEditor
                      variables={envs.map((entry, index) => ({
                        id: `template-${index}-${entry.name}`,
                        key: entry.name,
                        value: entry.value,
                        secret: /password|secret|key|token/i.test(entry.name),
                      }))}
                      onChange={(variables) =>
                        setEnvs(
                          variables.map((variable) => ({
                            name: variable.key,
                            value: variable.value,
                          })),
                        )
                      }
                    />
                    <p className="font-code-md text-[10px] text-on-surface-variant/50">
                      Secrets (password/token/key) are stored encrypted. Click
                      the refresh icon to regenerate a value.
                    </p>
                  </div>

                  <details className="group overflow-hidden rounded-2xl border border-outline-variant">
                    <summary className="flex items-center gap-sm px-md py-2.5 font-label-caps text-label-caps text-on-surface-variant uppercase cursor-pointer select-none hover:bg-surface-container-high/40 transition-colors">
                      <CaretDown
                        size={16}
                        className="transition-transform group-open:rotate-180"
                        aria-hidden="true"
                      />
                      Resource Allocation Matrix
                      <span className="ml-auto font-code-md text-[10px] text-on-surface-variant/70 normal-case">
                        {fmtCpu(cpu)} CPU · {ram} GB · {vol} VOL
                      </span>
                    </summary>
                    <div className="px-md pb-md pt-lg border-t border-outline-variant/60 flex flex-col gap-lg">
                      <div>
                        <div className="flex items-center justify-between mb-sm">
                          <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">
                            Compute Cores{" "}
                            <span className="text-on-surface-variant/50">
                              vCPU
                            </span>
                          </label>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                          {CPU_SEGS.map((c) => (
                            <button
                              key={c}
                              type="button"
                              onClick={() => setCpu(c)}
                              className={`${segmentButtonClass} ${cpu === c ? "border-primary bg-primary/15 text-primary" : ""}`}
                            >
                              {c}
                            </button>
                          ))}
                          <input
                            value={cpu}
                            onChange={(e) => setCpu(e.target.value)}
                            placeholder="Custom"
                            className="h-8 w-24 rounded-xl border border-border bg-surface-control px-2 py-1 font-code-md text-[11px] text-foreground outline-none transition-[background-color,border-color,box-shadow] duration-150 hover:bg-surface-container-highest/40 focus:border-primary focus:ring-2 focus:ring-ring/20"
                          />
                        </div>
                      </div>
                      <div>
                        <div className="flex items-center justify-between mb-sm">
                          <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">
                            Memory Allocation{" "}
                            <span className="text-on-surface-variant/50">
                              RAM
                            </span>
                          </label>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                          {RAM_SEGS.map((m) => (
                            <button
                              key={m}
                              type="button"
                              onClick={() => setRam(m)}
                              className={`${segmentButtonClass} ${ram === m ? "border-primary bg-primary/15 text-primary" : ""}`}
                            >
                              {m}
                            </button>
                          ))}
                          <input
                            value={ram}
                            onChange={(e) => setRam(e.target.value)}
                            placeholder="Custom"
                            className="h-8 w-24 rounded-xl border border-border bg-surface-control px-2 py-1 font-code-md text-[11px] text-foreground outline-none transition-[background-color,border-color,box-shadow] duration-150 hover:bg-surface-container-highest/40 focus:border-primary focus:ring-2 focus:ring-ring/20"
                          />
                        </div>
                      </div>
                      <div>
                        <div className="flex items-center justify-between mb-sm">
                          <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">
                            Volume Quota{" "}
                            <span className="text-on-surface-variant/50">
                              STORAGE MAX
                            </span>
                          </label>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                          {VOL_SEGS.map((v) => (
                            <button
                              key={v}
                              type="button"
                              onClick={() => setVol(v)}
                              className={`${segmentButtonClass} ${vol === v ? "border-primary bg-primary/15 text-primary" : ""}`}
                            >
                              {v}
                            </button>
                          ))}
                          <input
                            value={vol}
                            onChange={(e) => setVol(e.target.value)}
                            placeholder="Custom"
                            className="h-8 w-24 rounded-xl border border-border bg-surface-control px-2 py-1 font-code-md text-[11px] text-foreground outline-none transition-[background-color,border-color,box-shadow] duration-150 hover:bg-surface-container-highest/40 focus:border-primary focus:ring-2 focus:ring-ring/20"
                          />
                        </div>
                        <p className="font-code-md text-[10px] text-on-surface-variant/50 mt-sm">
                          Maximum quota applied to the persistent volume
                          (storage-opt size). 0 = unlimited.
                        </p>
                      </div>
                    </div>
                  </details>
                </>
              )}
            </div>
          )}
        </div>

        <div className="px-lg py-4 border-t border-outline-variant bg-surface-container-low/50 flex justify-between items-center shrink-0">
          <Button
            variant="ghost"
            onClick={() => (step === 2 ? setStep(1) : onClose())}
          >
            {step === 2 ? "BACK TO TEMPLATES" : "CANCEL"}
          </Button>
          {step === 2 && (
            <Button onClick={() => create()} disabled={creating}>
              Create Service
            </Button>
          )}
        </div>
      </div>
    </Modal>
  );
}
