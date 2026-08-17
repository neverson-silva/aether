import { useState } from "react";
import { useCreateDatabase, useProjects } from "../hooks";
import { useOverlayGate } from "./OverlayManager";
import { TechIcon } from "./TechIcon";
import { AdvancedSettings } from "./AdvancedSettings";
import { Button, Input, Select, useToast } from "./ui";

export const DB_ENGINES: { value: string; label: string; tagline: string; icon: string; versions: string[]; statA: [string, string]; statB: [string, string]; accent?: string }[] = [
  { value: "redis", label: "Redis", tagline: "In-Memory Datastore", icon: "redis", versions: ["7.0.12 (Stable)", "7.2.7", "7.4.2"], statA: ["OPS/SEC", "~100K+"], statB: ["LATENCY", "<1ms"] },
  { value: "postgres", label: "PostgreSQL", tagline: "Relational SQL", icon: "postgresql", versions: ["16 (Stable)", "15", "14"], statA: ["ACID", "Strict"], statB: ["USE CASE", "Primary DB"] },
  { value: "mongodb", label: "MongoDB", tagline: "NoSQL Document", icon: "mongodb", versions: ["7.0 (Stable)", "6.0", "7.0.16"], statA: ["SCHEMA", "Dynamic"], statB: ["USE CASE", "JSON / Scale"] },
  { value: "mysql", label: "MySQL", tagline: "Relational SQL", icon: "mysql", versions: ["8.4 (Stable)", "8.0", "8.4.3"], statA: ["ACID", "Strict"], statB: ["USE CASE", "Web Apps"] },
  { value: "mariadb", label: "MariaDB", tagline: "MySQL Compatible", icon: "mariadb", versions: ["11.4 (Stable)", "11.6", "10.11"], statA: ["ACID", "Strict"], statB: ["USE CASE", "Drop-in MySQL"] },
  { value: "mssql", label: "SQL Server", tagline: "Enterprise SQL", icon: "sqlserver", versions: ["2022 (Stable)", "2022-CU15", "2022-CU14"], statA: ["ACID", "Strict"], statB: ["USE CASE", "Enterprise"] },
  { value: "oracle", label: "Oracle", tagline: "Enterprise SQL", icon: "oracle", versions: ["23 (Stable)", "23.3", "23.2"], statA: ["ACID", "Strict"], statB: ["USE CASE", "Enterprise"] },
];

const MAIN_ENGINES = ["redis", "postgres", "mongodb"];

function dbImage(engine: string): string {
  const map: Record<string, string> = {
    redis: "docker.io/redis:7",
    postgres: "docker.io/postgres:16",
    mongodb: "docker.io/mongo:7",
    mysql: "docker.io/mysql:8.4",
    mariadb: "docker.io/mariadb:11",
    mssql: "mcr.microsoft.com/mssql/server:2022",
    oracle: "gvenzl/oracle-free:23",
  };
  return map[engine] ?? "docker.io/" + engine;
}

function StepBadge({ n, label, active }: { n: string; label: string; active: boolean }) {
  if (active) {
    return (
      <div className="relative flex items-center">
        <div className="absolute inset-0 bg-primary/20 blur-md rounded-full" />
        <div className="relative bg-[#050505] border border-primary/50 rounded-full px-5 py-2 flex items-center gap-3">
          <div className="w-1.5 h-1.5 rounded-full bg-primary shadow-[0_0_8px_rgba(176,198,255,0.8)]" />
          <span className="font-code-md text-sm text-primary font-medium tracking-wide">{n} // {label}</span>
        </div>
      </div>
    );
  }
  return (
    <div className="px-5 py-2 flex items-center gap-3 opacity-50">
      <div className="w-1.5 h-1.5 rounded-full bg-white/30" />
      <span className="font-code-md text-sm text-white/50 tracking-wide">{n} // {label}</span>
    </div>
  );
}

export function DatabaseWizard({ open, onClose, fixedProjectId, initialEngine }: { open: boolean; onClose: () => void; fixedProjectId?: string; initialEngine?: string }) {
  const initial = DB_ENGINES.find((e) => e.value === initialEngine) ?? DB_ENGINES[0];
  const { data: projects } = useProjects();
  const createDb = useCreateDatabase();
  const { toast } = useToast();

  const [step, setStep] = useState(1);
  const [engine, setEngine] = useState(initial.value);
  const [version, setVersion] = useState(initial.versions[0]);
  const [projectId, setProjectId] = useState(fixedProjectId ?? "");
  const [name, setName] = useState("");
  const [user, setUser] = useState("");
  const [password, setPassword] = useState("");
  const [memMB, setMemMB] = useState(512);
  const [storageMB, setStorageMB] = useState(0);
  const [creating, setCreating] = useState(false);
  const [versionOpen, setVersionOpen] = useState(false);

  const { mounted, closing, close } = useOverlayGate("db-wizard", open, onClose);

  if (!mounted) return null;

  const selected = DB_ENGINES.find((e) => e.value === engine)!;
  const moreEngines = DB_ENGINES.filter((e) => !MAIN_ENGINES.includes(e.value));

  const pickEngine = (value: string) => {
    const eng = DB_ENGINES.find((e) => e.value === value)!;
    setEngine(value);
    setVersion(eng.versions[0]);
    setVersionOpen(false);
  };

  const create = async () => {
    if (!projectId || !name.trim()) {
      toast("Select a project and set a name", "error");
      return;
    }
    setCreating(true);
    try {
      await createDb.mutateAsync({ project_id: projectId, name, engine, version, mem_mb: memMB, storage_mb: storageMB });
      toast("Deploy it manually from the service page", "info");
      onClose();
    } catch (err) {
      toast(err instanceof Error ? err.message : "failed to create database", "error");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className={`fixed inset-0 z-[80] flex items-center justify-center bg-black/70 p-4 ${closing ? "animate-fade-out" : "animate-fade-in"}`} onClick={() => close()}>
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Database wizard"
        onClick={(e) => e.stopPropagation()}
        className="glass-panel glow-border rounded-xl w-full max-w-[900px] max-h-[90vh] flex flex-col overflow-hidden relative z-10 animate-modal-pop"
      >
        <div className="absolute top-0 left-0 right-0 h-[1px] bg-gradient-to-r from-transparent via-primary to-transparent opacity-50" />

        <div className="px-4 md:px-8 pt-4 md:pt-8 pb-4 md:pb-6 border-b border-white/5 relative shrink-0">
          <div className="flex justify-between items-start mb-8">
            <div className="flex items-center gap-4">
              <div className="p-2.5 rounded-lg bg-primary/10 border border-primary/20 node-icon-glow">
                <span className="material-symbols-outlined text-primary text-[28px]" style={{ fontVariationSettings: "'FILL' 1" }}>dataset</span>
              </div>
              <div>
                <h1 className="font-headline-sm text-xl md:text-[28px] font-bold text-white tracking-tight mb-1">Deploy Database</h1>
                <p className="font-code-md text-xs text-on-surface-variant/80 uppercase tracking-[0.1em]">Node Configuration Phase</p>
              </div>
            </div>
            <button onClick={() => close()} className="text-on-surface-variant hover:text-white transition-colors p-2 rounded-lg hover:bg-white/5 focus:outline-none">
              <span className="material-symbols-outlined text-[24px]">close</span>
            </button>
          </div>
          <div className="flex items-center gap-4">
            <StepBadge n="01" label={step === 1 ? "Engine" : "Engine"} active={step >= 1} />
            <div className="h-[1px] w-8 bg-gradient-to-r from-primary/50 to-white/10" />
            <StepBadge n="02" label="Credentials" active={step === 2} />
          </div>
        </div>

        <div className="p-4 md:p-8 flex flex-col gap-8 bg-black/40 overflow-y-auto sidebar-scroll flex-1">
          {step === 1 && (
            <>
              <div className="flex flex-col gap-4">
                <div className="flex justify-between items-end">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Select Compute Engine</label>
                  <span className="font-code-md text-[10px] text-on-surface-variant/50">{DB_ENGINES.length} AVAILABLE NODES</span>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 md:gap-4">
                  {MAIN_ENGINES.map((key) => {
                    const eng = DB_ENGINES.find((e) => e.value === key)!;
                    const active = engine === eng.value;
                    return (
                      <div
                        key={eng.value}
                        onClick={() => pickEngine(eng.value)}
                        className={`node-card ${active ? "active" : ""} border ${active ? "border-primary/40" : "border-white/10"} rounded-lg p-3 md:p-5 cursor-pointer relative overflow-hidden ${active ? "" : "opacity-70 hover:opacity-100"}`}
                      >
                        {active && <div className="absolute top-0 right-0 w-24 h-24 bg-primary/10 rounded-full blur-2xl -mr-10 -mt-10" />}
                        <div className="flex justify-between items-start mb-4 relative z-10">
                          <TechIcon name={eng.icon} size={26} className={`node-icon-glow md:hidden ${active ? "text-primary" : "text-on-surface-variant"}`} />
                          <TechIcon name={eng.icon} size={32} className={`node-icon-glow hidden md:block ${active ? "text-primary" : "text-on-surface-variant"}`} />
                          <div className={`w-2 h-2 rounded-full ${active ? "bg-green-400 shadow-[0_0_8px_rgba(34,197,94,0.6)]" : "bg-white/20"}`} />
                        </div>
                        <div className="relative z-10 mb-4">
                          <h3 className="font-headline-sm text-base md:text-lg text-white mb-1">{eng.label}</h3>
                          <p className={`font-code-md text-[11px] ${active ? "text-primary/60" : "text-on-surface-variant"}`}>{eng.tagline}</p>
                        </div>
                        <div className="relative z-10 pt-4 border-t border-white/10 flex flex-col gap-2">
                          <div className="flex justify-between items-center">
                            <span className="font-code-md text-[10px] text-on-surface-variant">{eng.statA[0]}</span>
                            <span className="font-code-md text-[11px] text-white">{eng.statA[1]}</span>
                          </div>
                          <div className="flex justify-between items-center">
                            <span className="font-code-md text-[10px] text-on-surface-variant">{eng.statB[0]}</span>
                            <span className="font-code-md text-[11px] text-white">{eng.statB[1]}</span>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
                <div className="flex items-center gap-3 pt-1">
                  <span className="font-code-md text-[10px] text-on-surface-variant/50 tracking-widest">MORE ENGINES</span>
                  <div className="h-px flex-1 bg-white/5" />
                  <div className="flex gap-2">
                    {moreEngines.map((eng) => {
                      const active = engine === eng.value;
                      return (
                        <button
                          key={eng.value}
                          onClick={() => pickEngine(eng.value)}
                          className={`flex items-center gap-2 px-3 py-1.5 rounded-full border font-code-md text-[11px] transition-all ${active ? "border-primary/50 bg-primary/10 text-primary" : "border-white/10 text-on-surface-variant hover:border-primary/30 hover:text-white"}`}
                        >
                          <TechIcon name={eng.icon} size={12} />
                          {eng.label}
                        </button>
                      );
                    })}
                  </div>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-6 pt-4 border-t border-white/5">
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Image Version</label>
                  <div className="relative">
                    <button
                      onClick={() => setVersionOpen((o) => !o)}
                      className={`w-full bg-[#050505] border rounded-lg px-4 py-3.5 flex items-center justify-between text-left focus:outline-none transition-all hover:bg-white/5 group ${versionOpen ? "border-primary shadow-[0_0_15px_rgba(176,198,255,0.15)]" : "border-white/10 focus:border-primary"}`}
                    >
                      <div className="flex items-center gap-3">
                        <span className="font-code-md text-sm text-white group-hover:text-primary transition-colors">{version}</span>
                      </div>
                      <span className={`material-symbols-outlined text-white/50 text-[20px] transition-transform ${versionOpen ? "rotate-180 text-primary" : ""}`}>expand_more</span>
                    </button>
                    {versionOpen && (
                      <div className="absolute z-20 mt-1 w-full bg-[#0a0a0a] border border-white/10 rounded-lg overflow-hidden shadow-[0_8px_32px_rgba(0,0,0,0.6)]">
                        {selected.versions.map((v) => (
                          <button
                            key={v}
                            onClick={() => {
                              setVersion(v);
                              setVersionOpen(false);
                            }}
                            className={`w-full text-left px-4 py-2 font-code-md text-sm transition-colors ${version === v ? "text-primary bg-primary/10" : "text-white hover:bg-white/5"}`}
                          >
                            {v}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Target Artifact</label>
                  <div className="bg-[#050505] border border-white/10 rounded-lg p-3.5 flex items-center justify-between gap-4 h-[54px]">
                    <div className="flex items-center gap-2">
                      <span className="font-code-md text-xs font-semibold text-white">{dbImage(engine)}</span>
                      <span className="font-code-md text-[9px] text-[#22c55e] bg-[#22c55e]/10 border border-[#22c55e]/20 px-1.5 py-0.5 rounded uppercase tracking-wider">verified</span>
                    </div>
                    <span className="font-code-md text-[10px] text-on-surface-variant/70 text-right">Official Image</span>
                  </div>
                </div>
              </div>
            </>
          )}

          {step === 2 && (
            <div className="flex flex-col gap-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Project</label>
                  <Select value={projectId} onChange={(e) => setProjectId(e.target.value)} disabled={!!fixedProjectId}>
                    <option value="">Select project...</option>
                    {(projects ?? []).map((p) => (
                      <option key={p.id} value={p.id}>{p.name}</option>
                    ))}
                  </Select>
                </div>
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Database Name</label>
                  <Input icon="storage" placeholder="main-db" value={name} onChange={(e) => setName(e.target.value)} />
                </div>
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Database User</label>
                  <Input icon="person" placeholder="admin" value={user} onChange={(e) => setUser(e.target.value)} />
                </div>
                <div className="flex flex-col gap-3">
                  <label className="font-label-caps text-[10px] text-primary/70 tracking-[0.2em] uppercase">Password</label>
                  <Input icon="key" type="password" placeholder="generated if empty" value={password} onChange={(e) => setPassword(e.target.value)} />
                </div>
              </div>
              <AdvancedSettings
                showHealth={false}
                values={{ cpu: "0.5", memMB, storageMB, healthEnabled: false, healthPath: "/" }}
                onChange={(v) => {
                  setMemMB(v.memMB);
                  setStorageMB(v.storageMB);
                }}
              />
              <p className="font-code-md text-[11px] text-on-surface-variant/70">
                Credentials are encrypted and exposed to services of this project via environment variables.
              </p>
            </div>
          )}
        </div>

        <div className="px-4 md:px-8 py-4 md:py-5 border-t border-white/10 bg-black/60 backdrop-blur-xl flex justify-between items-center shrink-0">
          <button
            onClick={() => (step > 1 ? setStep(step - 1) : close())}
            className="px-8 py-3 font-label-caps text-xs text-white/60 hover:text-white transition-colors focus:outline-none tracking-widest border border-white/15 rounded-lg hover:border-white/40"
          >
            {step > 1 ? "BACK" : "CANCEL"}
          </button>
          {step === 1 ? (
            <button
              onClick={() => setStep(2)}
              className="metallic-btn px-8 py-3 font-label-caps text-xs text-on-primary rounded-lg transition-all focus:outline-none tracking-widest font-bold"
            >
              INITIALIZE NODE
            </button>
          ) : (
            <button
              onClick={() => create()}
              disabled={creating}
              className="metallic-btn px-8 py-3 font-label-caps text-xs text-on-primary rounded-lg transition-all focus:outline-none tracking-widest font-bold disabled:opacity-50"
            >
              {creating ? "CREATING..." : "CREATE DATABASE"}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
