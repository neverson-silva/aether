import { useEffect, useRef, useState } from "react";
import { Button, Modal, Spinner, useToast } from "./ui";
import { VariablePicker, type PickedVar } from "./VariablePicker";
import { useScopeVariables } from "./useScopeVariables";

const INTERP_HINT =
  "Use this syntax to reference environment-level variables: API_URL=${{environment.API_URL}} · SHARED_API=${{project.SHARED_API}}";

export interface EnvEntry {
  key: string;
  value: string;
  is_secret: boolean;
}

interface Row {
  key: string;
  value: string;
  secret: boolean;
  reveal: boolean;
  masked: boolean;
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
  const { toast } = useToast();
  const [rows, setRows] = useState<Row[]>([]);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [picker, setPicker] = useState<{ row: number; caret: number; query: string } | null>(null);
  const scopeGroups = useScopeVariables({
    serviceVars: (vars ?? []).map((v) => ({ key: v.key, value: v.value, secret: v.is_secret })),
  });
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);
  const maskTimers = useRef<Record<number, ReturnType<typeof setTimeout>>>({});

  useEffect(() => {
    return () => {
      Object.values(maskTimers.current).forEach(clearTimeout);
    };
  }, []);

  useEffect(() => {
    if (open && vars) {
      setRows(
        vars.map((v) => ({
          key: v.key,
          value: v.value,
          secret: v.is_secret,
          reveal: false,
          masked: true,
        }))
      );
      setError("");
    }
  }, [open, vars]);

  const setRow = (i: number, patch: Partial<Row>) => {
    setRows((prev) => prev.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  };

  const updateKey = (i: number, key: string) => {
    const clean = key.replace(/[^A-Za-z0-9_]/g, "").toUpperCase();
    setRow(i, { key: clean });
    setError("");
  };

  const addRow = () => {
    setRows((prev) => {
      const last = prev[prev.length - 1];
      if (last && !last.key && !last.value) return prev;
      return [...prev, { key: "", value: "", secret: false, reveal: false, masked: false }];
    });
  };

  const removeRow = (i: number) => {
    setRows((prev) => prev.filter((_, idx) => idx !== i));
  };

  const maskRow = (i: number) => {
    setRows((prev) => prev.map((r, idx) => (idx === i && !r.reveal ? { ...r, masked: true } : r)));
  };

  const scheduleMask = (i: number) => {
    if (maskTimers.current[i]) clearTimeout(maskTimers.current[i]);
    maskTimers.current[i] = setTimeout(() => maskRow(i), 900);
  };

  const parse = (): Record<string, { value: string; secret: boolean }> | null => {
    const entries: Record<string, { value: string; secret: boolean }> = {};
    for (let i = 0; i < rows.length; i++) {
      const r = rows[i];
      const key = r.key.trim();
      if (!key) continue;
      if (entries[key]) {
        setError(`Duplicate key: ${key}`);
        return null;
      }
      const value = r.value;
      entries[key] = { value, secret: r.secret };
    }
    setError("");
    return entries;
  };

  const save = async () => {
    const entries = parse();
    if (!entries) return;
    setSaving(true);
    try {
      const res = (await onSave(entries)) as { saved?: number } | void;
      toast(`Saved ${(res as { saved?: number })?.saved ?? Object.keys(entries).length} variable(s)`);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to save");
    } finally {
      setSaving(false);
    }
  };

  const bulkImport = () => {
    const pasted = window.prompt("Paste .env content (KEY=value per line):");
    if (!pasted) return;
    const lines = pasted.split("\n");
    const imported: Row[] = [];
    let failed = false;
    for (const raw of lines) {
      const line = raw.trim();
      if (!line || line.startsWith("#")) continue;
      const m = line.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
      if (!m) {
        setError(`Invalid line: ${line}`);
        failed = true;
        break;
      }
      imported.push({ key: m[1], value: m[2].replace(/^"|"$/g, ""), secret: /password|secret|key|token/i.test(m[1]), reveal: false, masked: true });
    }
    if (!failed) {
      setRows((prev) => [...prev.filter((r) => r.key), ...imported]);
      setError("");
    }
  };

  const handleValueChange = (i: number, val: string, el: HTMLInputElement) => {
    setRow(i, { value: val, masked: false });
    scheduleMask(i);
    const idx = val.lastIndexOf("${");
    if (idx >= 0) {
      const caret = el.selectionStart ?? val.length;
      setPicker({ row: i, caret: idx, query: val.slice(idx + 2, caret) });
    } else {
      setPicker(null);
    }
  };

  const insertVar = (v: PickedVar) => {
    if (!picker) return;
    const prev = rows[picker.row]?.value ?? "";
    const next = prev.slice(0, picker.caret) + "${" + v.name + "}" + prev.slice(picker.caret);
    const caret = picker.caret + "${".length + v.name.length + 1;
    const el = inputRefs.current[picker.row];
    setRow(picker.row, { value: next, masked: false });
    setPicker(null);
    requestAnimationFrame(() => {
      if (el) {
        el.focus();
        try {
          el.setSelectionRange(caret, caret);
        } catch {
        }
      }
    });
  };

  const count = rows.filter((r) => r.key.trim()).length;

  return (
    <Modal open={open} onClose={onClose} title={title} wide>
      <div className="space-y-md">
        <p className="font-body-sm text-body-sm text-on-surface-variant">{description}</p>
        <p className="bg-surface-container-lowest border border-outline-variant rounded-lg p-sm font-code-md text-code-md text-[11px] text-on-surface-variant/80">
          {INTERP_HINT}
        </p>

        {isLoading ? (
          <Spinner label="Loading variables..." />
        ) : (
          <>
            <div className="flex items-center justify-between">
              <span className="font-body-sm text-body-sm text-on-surface-variant">{count} variable{count === 1 ? "" : "s"}</span>
              <Button variant="subtle" size="sm" leftIcon="upload_file" onClick={bulkImport}>
                Bulk import (.env)
              </Button>
            </div>
            <div className="space-y-sm max-h-[48vh] overflow-y-auto sidebar-scroll pr-1">
              {rows.length === 0 && (
                <p className="font-body-sm text-body-sm text-on-surface-variant/60 text-center py-lg">No variables yet. Add one below.</p>
              )}
              {rows.map((r, i) => (
                <div key={i} className="flex gap-sm items-start">
                  <div className="flex-1">
                    <input
                      value={r.key}
                      onChange={(e) => updateKey(i, e.target.value)}
                      placeholder="KEY"
                      spellCheck={false}
                      className="w-full bg-surface border border-outline-variant rounded px-sm py-2 font-code-md text-code-md text-on-surface focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all placeholder:text-on-surface-variant/40 uppercase"
                    />
                  </div>
                  <div className="flex-1 relative">
                    <input
                      ref={(el) => {
                        if (inputRefs.current) inputRefs.current[i] = el;
                      }}
                      value={r.masked ? "••••••••••••" : r.value}
                      onChange={(e) => handleValueChange(i, e.target.value, e.target)}
                      onFocus={() => {
                        if (maskTimers.current[i]) clearTimeout(maskTimers.current[i]);
                        setRow(i, { masked: false });
                      }}
                      onBlur={() => maskRow(i)}
                      placeholder="VALUE"
                      spellCheck={false}
                      type="text"
                      className="w-full bg-surface border border-outline-variant rounded px-sm py-2 pr-9 font-code-md text-code-md text-on-surface focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all placeholder:text-on-surface-variant/40"
                    />
                    <button
                      type="button"
                      onClick={() => setRow(i, { reveal: !r.reveal, masked: r.reveal })}
                      className="absolute right-2 top-2 text-on-surface-variant hover:text-on-surface"
                      title={r.reveal ? "Hide value" : "Show value"}
                    >
                      <span className="material-symbols-outlined text-[18px]">{r.reveal ? "visibility_off" : "visibility"}</span>
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeRow(i)}
                    className="p-2 mt-1 text-on-surface-variant hover:text-error transition-colors"
                    title="Delete"
                  >
                    <span className="material-symbols-outlined text-[18px]">delete</span>
                  </button>
                </div>
              ))}
            </div>
            <button
              type="button"
              onClick={addRow}
              className="w-full py-2 border border-dashed border-outline-variant hover:border-primary text-on-surface-variant hover:text-primary text-body-sm font-body-sm rounded flex items-center justify-center gap-xs transition-colors"
            >
              <span className="material-symbols-outlined text-[18px]">add</span>
              Add Variable
            </button>
          </>
        )}
        {error && <p className="font-body-sm text-body-sm text-error">{error}</p>}
        {picker && scopeGroups.length > 0 && (
          <>
            <div className="fixed inset-0 z-[115]" onClick={() => setPicker(null)} />
            <VariablePicker
              groups={scopeGroups}
              query={picker.query}
              open
              onSelect={insertVar}
              onClose={() => setPicker(null)}
              inputRef={inputRefs as unknown as React.RefObject<HTMLInputElement>}
              previewText="Resolved at deploy time."
            />
          </>
        )}
        <div className="flex justify-end gap-md">
          <Button type="button" variant="ghost" onClick={onClose}>Close</Button>
          <Button onClick={save} disabled={saving || isLoading}>
            <span className="material-symbols-outlined text-[16px]">save</span>
            Save variables
          </Button>
        </div>
      </div>
    </Modal>
  );
}
