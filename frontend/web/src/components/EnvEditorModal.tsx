import { useEffect, useMemo, useState } from "react";
import { Button, Dialog, Skeleton, VariableEditor, type VariableRow, useToast } from "@aether/design-system";

export interface EnvEntry {
  key: string;
  value: string;
  is_secret: boolean;
}

function parseEnv(): VariableRow[] | undefined {
  const pasted = window.prompt("Paste .env content (KEY=value per line):");
  if (!pasted) return;
  return pasted.split("\n").flatMap((raw, index) => {
    const line = raw.trim();
    if (!line || line.startsWith("#")) return [];
    const match = line.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
    if (!match) return [];
    return [{
      id: `imported-${index}-${match[1]}`,
      key: match[1],
      value: match[2].replace(/^"|"$/g, ""),
      secret: /password|secret|key|token/i.test(match[1]),
    }];
  });
}

export function EnvEditorModal({
  open,
  onClose,
  title,
  description,
  isLoading,
  vars,
  onSave,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  description: string;
  isLoading?: boolean;
  vars?: EnvEntry[];
  onSave: (entries: Record<string, { value: string; secret: boolean }>) => Promise<unknown> | unknown;
}) {
  const { add } = useToast();
  const [variables, setVariables] = useState<VariableRow[]>([]);
  const variableKey = useMemo(() => `${open}-${(vars ?? []).map((entry) => `${entry.key}:${entry.is_secret}`).join("|")}`, [open, vars]);

  useEffect(() => {
    if (open && vars) {
      setVariables(vars.map((entry, index) => ({
        id: `service-${index}-${entry.key}`,
        key: entry.key,
        value: entry.value,
        secret: entry.is_secret,
      })));
    }
  }, [open, vars]);

  const save = async () => {
    const entries: Record<string, { value: string; secret: boolean }> = {};
    for (const variable of variables) {
      const key = variable.key.trim();
      if (key) entries[key] = { value: variable.value, secret: Boolean(variable.secret) };
    }
    try {
      const result = await onSave(entries) as { saved?: number } | void;
      add({ title: "Variables saved", description: `${result?.saved ?? Object.keys(entries).length} variable(s) updated.`, tone: "success" });
    } catch (error) {
      add({ title: "Variables could not be saved", description: error instanceof Error ? error.message : "Try again later.", tone: "error" });
    }
  };

  const exportVariables = () => {
    const content = variables.filter((variable) => variable.key.trim()).map((variable) => `${variable.key}=${variable.value}`).join("\n");
    void navigator.clipboard?.writeText(content);
    add({ title: "Variables copied", description: "Environment variables were copied as .env text.", tone: "success" });
  };

  return (
    <Dialog open={open} onOpenChange={(value) => { if (!value) onClose(); }} title={title} description={description} trigger={<button type="button" className="hidden" aria-hidden="true" tabIndex={-1} />} footer={(
      <div className="flex w-full justify-end gap-md pr-2">
        <Button type="button" variant="ghost" onClick={onClose}>Close</Button>
        <Button type="button" onClick={save} loading={isLoading} loadingLabel="Saving variables">Save variables</Button>
      </div>
    )}>
      {isLoading ? (
        <Skeleton variant="card" className="min-h-24" />
      ) : (
        <VariableEditor
          key={variableKey}
          variables={variables}
          onChange={setVariables}
          onImport={parseEnv}
          onExport={exportVariables}
          className="max-h-[min(34rem,calc(100dvh-14rem))]"
        />
      )}
    </Dialog>
  );
}
