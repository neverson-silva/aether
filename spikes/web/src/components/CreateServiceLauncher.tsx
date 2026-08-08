import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { ApplicationWizard } from "./ApplicationWizard";
import { DatabaseWizard } from "./DatabaseWizard";
import { ComposeWizard } from "./ComposeWizard";
import { useOverlay, useOverlayGate } from "./OverlayManager";
import { TechIcon } from "./TechIcon";
import { TemplateWizard } from "./TemplateWizard";

type Pending =
  | { kind: "app"; kind2?: "web" | "api" }
  | { kind: "db"; engine?: string }
  | { kind: "compose" }
  | { kind: "templates" };

export function CreateServiceLauncher({ open, onClose, fixedProjectId }: { open: boolean; onClose: () => void; fixedProjectId?: string }) {
  const [query, setQuery] = useState("");
  const [index, setIndex] = useState(0);
  const [pending, setPending] = useState<Pending | null>(null);
  const [wizardOpen, setWizardOpen] = useState(false);
  const navigate = useNavigate();

  const { active, close: requestClose } = useOverlay("launcher-create");
  const { mounted: launcherMounted, closing: launcherClosing } = useOverlayGate("launcher-create", open, onClose);

  useEffect(() => {
    if (launcherMounted) {
      setQuery("");
      setIndex(0);
    }
  }, [launcherMounted]);

  // quando o launcher terminou de fechar e há uma escolha pendente, abre o wizard
  useEffect(() => {
    if (pending && !active && !launcherMounted) {
      setWizardOpen(true);
    }
  }, [pending, active, launcherMounted]);

  const items = useMemo(() => {
    const q = query.trim().toLowerCase();
    const mk = (section: string, label: string, icon: string, hint: string, action: () => void) => ({
      id: section + label,
      section,
      label,
      icon,
      hint,
      action,
      match: !q || label.toLowerCase().includes(q),
    });
    const out: ReturnType<typeof mk>[] = [
      mk("Services", "Web (Frontend)", "react", "React, Next.js, Vite and other frontend frameworks", () => select({ kind: "app", kind2: "web" })),
      mk("Services", "API (Backend)", "go", "Go, Node, Python, FastAPI and other backends", () => select({ kind: "app", kind2: "api" })),
      mk("Services", "Database", "postgresql", "Provision a managed database", () => select({ kind: "db" })),
      mk("Services", "Compose (Docker Compose)", "docker", "Multi-container stack with compose file", () => select({ kind: "compose" })),
      mk("Services", "Browse Templates", "supabase", "One-click templates with default envs", () => select({ kind: "templates" })),
    ];
    return out.filter((i) => i.match);
  }, [query]);
  useEffect(() => {
    setIndex(0);
  }, [query]);

  const select = (choice: Pending) => {
    setPending(choice);
    requestClose();
  };

  const finishWizard = () => {
    setWizardOpen(false);
    setPending(null);
  };

  if (wizardOpen && pending) {
    if (pending.kind === "db") {
      return <DatabaseWizard open={wizardOpen} onClose={finishWizard} fixedProjectId={fixedProjectId} initialEngine={pending.engine} />;
    }
    if (pending.kind === "compose") {
      return <ComposeWizard open={wizardOpen} onClose={finishWizard} fixedProjectId={fixedProjectId} />;
    }
    if (pending.kind === "templates") {
      return <TemplateWizard open={wizardOpen} onClose={finishWizard} />;
    }
    return (
      <ApplicationWizard
        open={wizardOpen}
        onClose={finishWizard}
        fixedProjectId={fixedProjectId}
        kind={pending.kind2 ?? "web"}
      />
    );
  }

  if (!open && !launcherMounted) return null;

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setIndex((i) => Math.min(i + 1, items.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter" && items[index]) {
      items[index].action();
    }
  };

  return (
    <div
      className={`fixed inset-0 z-[85] flex items-start justify-center pt-[16vh] bg-black/60 backdrop-blur-[2px] ${launcherClosing ? "animate-fade-out" : "animate-fade-in"}`}
      onClick={requestClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Create service launcher"
        aria-live="polite"
        onClick={(e) => e.stopPropagation()}
        className={`w-full max-w-[560px] bg-surface-modal border border-outline-variant rounded-2xl shadow-xl overflow-hidden ${launcherClosing ? "animate-fade-out-up" : "animate-modal-pop"}`}
      >
        <div className="flex items-center gap-sm px-md py-3.5 border-b border-outline-variant">
          <span className="material-symbols-outlined text-[18px] text-on-surface-variant">add_circle</span>
          <input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="What do you want to create?"
            className="flex-1 bg-transparent font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant/50 focus:outline-none"
          />
          <kbd className="font-code-md text-code-md text-on-surface-variant/60 border border-outline-variant rounded px-1.5">esc</kbd>
        </div>
        <div className="max-h-[420px] overflow-y-auto py-1.5 sidebar-scroll">
          {items.length === 0 && <p className="px-md py-3 font-body-sm text-body-sm text-on-surface-variant">Nothing matches "{query}".</p>}
          {items.map((item, i) => (
            <div key={item.id} className="px-2">
              {(i === 0 || item.section !== items[i - 1]?.section) && (
                <p className="px-2.5 pt-2 pb-1 font-label-caps text-label-caps text-on-surface-variant/60 uppercase">{item.section}</p>
              )}
              <button
                onMouseEnter={() => setIndex(i)}
                onClick={item.action}
                className={`w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-colors ${
                  i === index ? "bg-surface-container-high" : "hover:bg-surface-container-high/60"
                }`}
              >
                <span className="shrink-0 w-6 flex justify-center"><TechIcon name={item.icon} size={18} className="text-on-surface-variant" /></span>
                <span className="flex-1 min-w-0">
                  <span className="block font-body-md text-body-md text-on-surface truncate">{item.label}</span>
                  <span className="block font-code-md text-code-md text-on-surface-variant/60 truncate">{item.hint}</span>
                </span>
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant/40">chevron_right</span>
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
