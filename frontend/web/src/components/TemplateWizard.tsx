import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { ArrowRight, CaretDown, Check, List, MagnifyingGlass, NotePencil, ShoppingBag } from "@phosphor-icons/react";
import { Button, Input, Modal, NativeSelect, VariableEditor, useToast } from "@aether/design-system";
import { useCreateApp, useInstallTemplate, useProjects, useTemplateCategories, useTemplatesFiltered } from "../hooks";
import type { TemplateItem } from "../hooks";
import { TechIcon } from "./TechIcon";

interface ParsedDef {
  services: { name: string; image: string; port: number; versions?: string[]; env?: Record<string, string> }[];
}

function parseDef(t: TemplateItem): ParsedDef {
  try {
    return JSON.parse(t.definition || "{}") as ParsedDef;
  } catch {
    return { services: [] };
  }
}

function genPassword(len = 24): string {
  const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
  let out = "";
  const arr = new Uint32Array(len);
  crypto.getRandomValues(arr);
  for (let i = 0; i < len; i++) out += chars[arr[i] % chars.length];
  return out;
}

function gen64(): string {
  return genPassword(32) + genPassword(32);
}

function resolveValue(v: string): string {
  if (v === "{{password}}") return genPassword();
  if (v === "{{random64}}") return gen64();
  return v;
}

function iconBg(t: TemplateItem): string {
  const cat = t.category.toLowerCase();
  if (cat === "database" || cat === "cache") return "rgba(86,141,255,0.1)";
  if (cat === "monitoring" || cat === "logging" || cat === "analytics") return "rgba(192,193,255,0.1)";
  if (cat === "ai" || cat === "automation") return "rgba(243,100,32,0.1)";
  if (cat === "security" || cat === "identity") return "rgba(140,144,161,0.1)";
  return "rgba(176,198,255,0.1)";
}
function iconColor(t: TemplateItem): string {
  const cat = t.category.toLowerCase();
  if (cat === "database" || cat === "cache") return "#568dff";
  if (cat === "monitoring" || cat === "logging" || cat === "analytics") return "#c0c1ff";
  if (cat === "ai" || cat === "automation") return "#f36420";
  if (cat === "security" || cat === "identity") return "#8c90a1";
  return "#b0c6ff";
}

const CPU_SEGS = [".25", ".5", "1", "2", "4", "8"];
const RAM_SEGS = [".25", ".5", "1", "2", "4", "8", "∞"];
const VOL_SEGS = ["1G", "5G", "10G", "50G", "100G", "∞"];

function fmtCpu(c: string): string {
  return c.startsWith(".") ? "0" + c : c;
}

export function TemplateWizard({ open, onClose }: { open: boolean; onClose: () => void }) {
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
  const [cpu, setCpu] = useState(".5");
  const [ram, setRam] = useState(".5");
  const [vol, setVol] = useState("∞");
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    if (open) {
      setStep(1);
      setSelected(null);
      setQuery("");
      setCategory("");
    }
  }, [open]);

  const { data: templates } = useTemplatesFiltered({ category: category || undefined, q: query || undefined });

  const pick = (t: TemplateItem) => {
    const def = parseDef(t);
    const svc = def.services[0];
    setSelected(t);
    setVersion(svc?.versions?.[0] ?? "");
    setStep(2);
    setName(t.name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") + "-" + Math.floor(Math.random() * 900 + 100));
    const envList: { name: string; value: string }[] = [];
    if (svc?.env) {
      for (const [k, v] of Object.entries(svc.env)) {
        envList.push({ name: k, value: resolveValue(v) });
      }
    }
    setEnvs(envList);
  };

  const envCount = (t: TemplateItem): number => {
    const svc = parseDef(t).services[0];
    return svc?.env ? Object.keys(svc.env).length : 0;
  };

  const create = async () => {
    if (!selected || !projectId || !name.trim()) {
      add({ title: "Select a project and set a name", tone: "error" });
      return;
    }
    setCreating(true);
    try {
      if (selected.compose_yaml) {
        await installTemplate.mutateAsync({ id: selected.id, project_id: projectId, name });
        add({ title: "Deploy it manually from the services page", tone: "info" });
        onClose();
        navigate({ to: "/apps" } as never);
        return;
      }
      const svc = parseDef(selected).services[0];
      const imageRef = version ? `${svc?.image}:${version}` : svc?.image ?? selected.name;
      const memMB = ram === "∞" ? 0 : Math.round(parseFloat(fmtCpu(ram)) * 1024);
      const storageMB = vol === "∞" ? 0 : Math.round(parseFloat(vol.replace("G", "")) * 1024);
      const app = await createApp.mutateAsync({
        projectID: projectId,
        payload: {
          name,
          source_type: "image",
          image: imageRef,
          port: svc?.port ?? 80,
          build_type: "dockerfile",
          resources: { cpus: fmtCpu(cpu), mem_mb: memMB, storage_mb: storageMB },
          health_check: { enabled: false, path: "/", interval_ms: 5000, timeout_ms: 2000, retries: 3 },
          env: envs.filter((e) => e.name.trim()).map((e) => ({ name: e.name, value: e.value, secret: /password|secret|key|token/i.test(e.name) })),
        } as never,
      });
      add({ title: "Deploy it manually from the service page", tone: "info" });
      onClose();
      navigate({ to: "/apps/$appId", params: { appId: app.id } } as never);
    } catch (err) {
      add({ title: err instanceof Error ? err.message : "Failed to create service", tone: "error" });
    } finally {
      setCreating(false);
    }
  };

  const svcOfSelected = selected ? parseDef(selected).services[0] : null;
  const isCompose = !!selected?.compose_yaml;

  return (
    <Modal open={open} onOpenChange={(value) => { if (!value) onClose(); }} title={step === 1 ? "Templates · Choose one to configure" : `${selected?.name} · Configure`} description={step === 1 ? "One-click templates with default environment variables, ready to configure and deploy." : "Pre-filled with the template's default environment variables — adjust as needed."} size="wizard">
      <div className="max-h-[75vh] overflow-y-auto">

        <div className="px-lg py-md border-b border-surface-variant flex flex-col gap-sm shrink-0" style={{ borderColor: "#353534" }}>
          <div className="flex justify-between items-start">
            <div>
              <h1 className="font-headline-sm text-headline-sm text-on-surface flex items-center gap-sm">
                <ShoppingBag size={24} className="text-tertiary-container" aria-hidden="true" />
                {step === 1 ? "Templates · Choose one to configure" : `${selected?.name} · Configure`}
              </h1>
              <p className="font-body-md text-body-md text-on-surface-variant mt-1">
                {step === 1
                  ? "One-click templates with default environment variables, ready to configure and deploy."
                  : "Pre-filled with the template's default environment variables — adjust as needed."}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 mt-2 font-label-caps text-label-caps">
            <div className={`flex items-center gap-1 ${step === 1 ? "text-primary" : "text-on-surface-variant"}`}>
              <div className={`w-5 h-5 rounded-full flex items-center justify-center border ${step === 1 ? "bg-primary/20 border-primary/30" : "border-surface-variant"}`}>{step === 1 ? "1" : "✓"}</div>
              <span>Browse</span>
            </div>
            <div className="w-8 h-[1px]" style={{ background: "#353534" }} />
            <div className={`flex items-center gap-1 ${step === 2 ? "text-primary" : "text-on-surface-variant"}`}>
              <div className="w-5 h-5 rounded-full border border-surface-variant flex items-center justify-center">2</div>
              <span>Configure</span>
            </div>
          </div>
        </div>

        <div className="px-lg py-md border-b flex items-center gap-md shrink-0" style={{ borderColor: "#353534", background: "rgba(19,19,19,0.5)" }}>
          <div className="relative w-full md:w-64">
            <MagnifyingGlass size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant" aria-hidden="true" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search templates..."
              className="w-full bg-surface-container-low border rounded-md font-body-sm text-body-sm text-on-surface pl-9 pr-3 py-1.5 focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/50 transition-all"
              style={{ borderColor: "#353534" }}
            />
          </div>
          <div className="relative">
            <NativeSelect label="" value={category} onChange={(e) => setCategory(e.target.value)} options={[{ label: "All categories", value: "" }, ...(categories ?? []).map((c) => ({ label: c, value: c }))]} className="w-56" />
            <CaretDown size={16} className="absolute right-2 top-1/2 -translate-y-1/2 text-on-surface-variant pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-lg scrollbar-hide">
          {step === 1 ? (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {(templates ?? []).map((t) => (
                  <button
                    key={t.id}
                    onClick={() => pick(t)}
                    className="bg-surface-container-low border border-surface-variant rounded-lg p-md flex flex-col justify-between h-[180px] card-hover transition-all cursor-pointer group relative"
                    style={{ borderColor: "#353534" }}
                  >
                    {t.featured && (
                      <div className="absolute top-2 right-2 bg-tertiary-container/20 text-tertiary-container font-label-caps text-[9px] px-1.5 py-0.5 rounded">FEATURED</div>
                    )}
                    <div className="flex flex-col items-center text-center">
                      <div className="w-14 h-14 rounded-xl flex items-center justify-center mb-3" style={{ background: iconBg(t), color: iconColor(t) }}>
                        <TechIcon name={t.icon} size={30} className="" />
                      </div>
                      <h3 className="font-body-md text-body-md font-semibold text-on-surface truncate w-full">{t.name}</h3>
                      <p className="font-body-sm text-body-sm text-on-surface-variant line-clamp-2 mt-1">{t.description}</p>
                    </div>
                    <div className="flex justify-between items-center mt-4">
                      <div className="flex items-center gap-1 text-on-surface-variant font-label-caps text-label-caps">
                        <List size={12} aria-hidden="true" /> {envCount(t)} env{envCount(t) === 1 ? "" : "s"}
                      </div>
                      <span className="font-label-caps text-label-caps text-primary opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                        Choose <ArrowRight size={12} aria-hidden="true" />
                      </span>
                    </div>
                  </button>
                ))}
              </div>
              {(templates ?? []).length === 0 && (
                <p className="font-body-sm text-body-sm text-on-surface-variant py-md text-center">No templates match "{query}".</p>
              )}
            </>
          ) : (
            <div className="p-lg flex flex-col gap-lg">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-md">
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Project</label>
                  <div className="relative">
                    <NativeSelect label="" value={projectId} onChange={(e) => setProjectId(e.target.value)} options={[{ label: "Select project...", value: "" }, ...(projects ?? []).map((p) => ({ label: p.name, value: p.id }))]} />
                    <CaretDown size={18} className="absolute right-2.5 top-2.5 text-outline pointer-events-none" aria-hidden="true" />
                  </div>
                </div>
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Name</label>
                  <div className="relative">
                    <Input placeholder="my-service" value={name} onChange={(e) => setName(e.target.value)} />
                  </div>
                </div>
              </div>

              {isCompose ? (
                <div className="flex flex-col gap-3">
                  <p className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Managed stack</p>
                  <div className="bg-[#050505] border border-white/10 rounded-lg p-md">
                    <p className="font-body-sm text-body-sm text-on-surface-variant">
                      This template deploys a full multi-service stack using its official Docker Compose (application + database + cache). Just pick a project and a name — the stack runs as-is. Start it from the services page afterwards.
                    </p>
                  </div>
                </div>
              ) : (
                <>
              <div className="flex flex-col gap-3">
                <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Version</label>
                <div className="flex flex-wrap items-center gap-2">
                  {(svcOfSelected?.versions ?? []).map((v) => (
                    <button
                      key={v}
                      onClick={() => setVersion(v)}
                      className={`segment-btn ${version === v ? "active" : ""}`}
                    >
                      {v}
                    </button>
                  ))}
                  <span className="font-code-md text-[10px] text-on-surface-variant/70 ml-sm">
                    Image: {svcOfSelected?.image}:{version || "?"}
                  </span>
                </div>
                <p className="font-code-md text-[10px] text-on-surface-variant/50">Versions are pinned (never :latest) for reproducible deployments.</p>
              </div>

              <div className="flex flex-col gap-3">
                <p className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Environment variables (defaults from template)</p>
                <VariableEditor
                  variables={envs.map((entry, index) => ({
                    id: `template-${index}-${entry.name}`,
                    key: entry.name,
                    value: entry.value,
                    secret: /password|secret|key|token/i.test(entry.name),
                  }))}
                  onChange={(variables) => setEnvs(variables.map((variable) => ({ name: variable.key, value: variable.value })))}
                />
                <p className="font-code-md text-[10px] text-on-surface-variant/50">
                  Secrets (password/token/key) are stored encrypted. Click the refresh icon to regenerate a value.
                </p>
              </div>

              <details className="group border border-outline-variant rounded-lg">
                <summary className="flex items-center gap-sm px-md py-2.5 font-label-caps text-label-caps text-on-surface-variant uppercase cursor-pointer select-none hover:bg-surface-container-high/40 transition-colors">
                  <CaretDown size={16} className="transition-transform group-open:rotate-180" aria-hidden="true" />
                  Resource Allocation Matrix
                  <span className="ml-auto font-code-md text-[10px] text-on-surface-variant/70 normal-case">{fmtCpu(cpu)} CPU · {ram} GB · {vol} VOL</span>
                </summary>
                <div className="px-md pb-md pt-lg border-t border-outline-variant/60 flex flex-col gap-lg">
                  <div>
                    <div className="flex items-center justify-between mb-sm">
                      <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Compute Cores <span className="text-on-surface-variant/50">vCPU</span></label>
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {CPU_SEGS.map((c) => (
                        <button key={c} onClick={() => setCpu(c)} className={`segment-btn ${cpu === c ? "active" : ""}`}>{c}</button>
                      ))}
                      <input
                        value={cpu}
                        onChange={(e) => setCpu(e.target.value)}
                        placeholder="Custom"
                        className="form-input-dark !py-1 !px-2 w-24 font-code-md text-[11px]"
                      />
                    </div>
                  </div>
                  <div>
                    <div className="flex items-center justify-between mb-sm">
                      <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Memory Allocation <span className="text-on-surface-variant/50">RAM</span></label>
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {RAM_SEGS.map((m) => (
                        <button key={m} onClick={() => setRam(m)} className={`segment-btn ${ram === m ? "active" : ""}`}>{m}</button>
                      ))}
                      <input
                        value={ram}
                        onChange={(e) => setRam(e.target.value)}
                        placeholder="Custom"
                        className="form-input-dark !py-1 !px-2 w-24 font-code-md text-[11px]"
                      />
                    </div>
                  </div>
                  <div>
                    <div className="flex items-center justify-between mb-sm">
                      <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Volume Quota <span className="text-on-surface-variant/50">STORAGE MAX</span></label>
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {VOL_SEGS.map((v) => (
                        <button key={v} onClick={() => setVol(v)} className={`segment-btn ${vol === v ? "active" : ""}`}>{v}</button>
                      ))}
                      <input
                        value={vol}
                        onChange={(e) => setVol(e.target.value)}
                        placeholder="Custom"
                        className="form-input-dark !py-1 !px-2 w-24 font-code-md text-[11px]"
                      />
                    </div>
                    <p className="font-code-md text-[10px] text-on-surface-variant/50 mt-sm">Maximum quota applied to the persistent volume (storage-opt size). 0 = unlimited.</p>
                  </div>
                </div>
              </details>
                </>
              )}
            </div>
          )}
        </div>

        <div className="px-lg py-4 border-t border-outline-variant bg-surface-container-low/50 flex justify-between items-center shrink-0">
          <Button variant="ghost" onClick={() => (step === 2 ? setStep(1) : onClose())}>
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
