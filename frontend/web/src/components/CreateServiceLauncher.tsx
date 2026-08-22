import { useState } from "react";
import { Code, Cube, Database, Globe, Stack } from "@phosphor-icons/react";
import { CommandPalette, type CommandPaletteItem } from "@aether/design-system";
import { ApplicationWizard } from "./ApplicationWizard";
import { ComposeWizard } from "./ComposeWizard";
import { DatabaseWizard } from "./DatabaseWizard";
import { TemplateWizard } from "./TemplateWizard";

type Pending =
  | { kind: "app"; kind2?: "web" | "api" }
  | { kind: "db"; engine?: string }
  | { kind: "compose" }
  | { kind: "templates" };

export function CreateServiceLauncher({ open, onClose, fixedProjectId, fixedEnvironmentId }: { open: boolean; onClose: () => void; fixedProjectId?: string; fixedEnvironmentId?: string }) {
  const [pending, setPending] = useState<Pending | null>(null);
  const select = (choice: Pending) => {
    setPending(choice);
    onClose();
  };

  const items: CommandPaletteItem[] = [
    { id: "web", label: "Web application", description: "React, Next.js, Vite and frontend frameworks", icon: <Globe size={20} weight="duotone" />, onSelect: () => select({ kind: "app", kind2: "web" }) },
    { id: "api", label: "API service", description: "Go, Node, Python and backend runtimes", icon: <Code size={20} weight="duotone" />, onSelect: () => select({ kind: "app", kind2: "api" }) },
    { id: "database", label: "Database", description: "Provision a managed database", icon: <Database size={20} weight="duotone" />, onSelect: () => select({ kind: "db" }) },
    { id: "compose", label: "Compose stack", description: "Deploy a multi-container Docker Compose stack", icon: <Stack size={20} weight="duotone" />, onSelect: () => select({ kind: "compose" }) },
    { id: "templates", label: "Browse templates", description: "Start with a ready-made service template", icon: <Cube size={20} weight="duotone" />, onSelect: () => select({ kind: "templates" }) },
  ];

  const finishWizard = () => {
    setPending(null);
    onClose();
  };

  if (pending?.kind === "db") return <DatabaseWizard open onClose={finishWizard} fixedProjectId={fixedProjectId} initialEngine={pending.engine} />;
  if (pending?.kind === "compose") return <ComposeWizard open onClose={finishWizard} fixedProjectId={fixedProjectId} />;
  if (pending?.kind === "templates") return <TemplateWizard open onClose={finishWizard} />;
  if (pending?.kind === "app") return <ApplicationWizard open onClose={finishWizard} fixedProjectId={fixedProjectId} fixedEnvironmentId={fixedEnvironmentId} kind={pending.kind2 ?? "web"} />;

  return (
    <CommandPalette
      open={open}
      onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}
      placeholder="Search or create..."
      empty="No service type matches your search."
      items={items}
    />
  );
}
