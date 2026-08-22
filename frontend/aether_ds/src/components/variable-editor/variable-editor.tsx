import { Eye, EyeSlash, Plus, Trash } from '@phosphor-icons/react'
import { useState } from 'react'
export interface VariableRow {
  id: string
  key: string
  value: string
  secret?: boolean
  scope?: string
}
export interface VariableEditorProps {
  variables: VariableRow[]
  onChange?: (variables: VariableRow[]) => void
  onImport?: () => VariableRow[] | void
  onExport?: () => void
  onBulkEdit?: (variables: VariableRow[]) => void
}
export function VariableEditor({
  onChange,
  onExport,
  onImport,
  onBulkEdit,
  variables: initial,
}: VariableEditorProps) {
  const [variables, setVariables] = useState(initial)
  const [revealed, setRevealed] = useState<string[]>([])
  const update = (next: VariableRow[]) => {
    setVariables(next)
    onChange?.(next)
  }
  const duplicates = new Set(
    variables
      .filter(
        (item, index) =>
          item.key &&
          variables.findIndex((candidate) => candidate.key === item.key) !==
            index,
      )
      .map((item) => item.id),
  )
  return (
    <section className="overflow-hidden rounded-lg border border-border">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-border p-3">
        <span className="text-body-sm font-semibold">Variables</span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => {
              const imported = onImport?.()
              if (imported) update(imported)
            }}
            className="text-body-sm text-muted-foreground hover:text-foreground"
          >
            Import
          </button>
          {onBulkEdit ? (
            <button
              type="button"
              onClick={() => onBulkEdit(variables)}
              className="text-body-sm text-muted-foreground hover:text-foreground"
            >
              Bulk edit
            </button>
          ) : null}
          <button
            type="button"
            onClick={onExport}
            className="text-body-sm text-muted-foreground hover:text-foreground"
          >
            Export
          </button>
          <button
            type="button"
            onClick={() =>
              update([
                ...variables,
                { id: crypto.randomUUID(), key: '', value: '' },
              ])
            }
            className="inline-flex items-center gap-1 rounded-md bg-primary px-2 py-1 text-body-sm text-primary-foreground"
          >
            <Plus size={14} />
            Add
          </button>
        </div>
      </header>
      <div className="divide-y divide-border">
        {variables.map((variable, index) => (
          <div
            key={variable.id}
            className="grid min-w-0 grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)_minmax(0,1fr)_auto] gap-2 p-3"
          >
            <input
              aria-label="Variable key"
              value={variable.key}
              onChange={(event) =>
                update(
                  variables.map((item, itemIndex) =>
                    itemIndex === index
                      ? { ...item, key: event.target.value }
                      : item,
                  ),
                )
              }
              placeholder="KEY"
              className="h-9 rounded-md border border-border bg-surface-background px-2 font-mono text-code-md outline-none focus:border-primary"
            />
            {duplicates.has(variable.id) ? (
              <span className="text-label-caps text-status-danger md:col-span-3">
                Duplicate key
              </span>
            ) : null}
            <input
              aria-label="Variable value"
              type={
                variable.secret && !revealed.includes(variable.id)
                  ? 'password'
                  : 'text'
              }
              value={variable.value}
              onChange={(event) =>
                update(
                  variables.map((item, itemIndex) =>
                    itemIndex === index
                      ? { ...item, value: event.target.value }
                      : item,
                  ),
                )
              }
              placeholder="Value"
              className="h-9 rounded-md border border-border bg-surface-background px-2 font-mono text-code-md outline-none focus:border-primary"
            />
            <input
              aria-label="Variable scope"
              value={variable.scope ?? ''}
              onChange={(event) =>
                update(
                  variables.map((item, itemIndex) =>
                    itemIndex === index
                      ? { ...item, scope: event.target.value }
                      : item,
                  ),
                )
              }
              placeholder="Scope"
              className="h-9 rounded-md border border-border bg-surface-background px-2 text-body-sm outline-none focus:border-primary"
            />
            {variable.secret ? (
              <button
                type="button"
                aria-label="Toggle secret visibility"
                onClick={() =>
                  setRevealed(
                    revealed.includes(variable.id)
                      ? revealed.filter((id) => id !== variable.id)
                      : [...revealed, variable.id],
                  )
                }
                className="text-muted-foreground"
              >
                {revealed.includes(variable.id) ? (
                  <EyeSlash size={18} />
                ) : (
                  <Eye size={18} />
                )}
              </button>
            ) : (
              <button
                type="button"
                aria-label="Remove variable"
                onClick={() =>
                  update(variables.filter((item) => item.id !== variable.id))
                }
                className="text-muted-foreground hover:text-status-danger"
              >
                <Trash size={18} />
              </button>
            )}
          </div>
        ))}
      </div>
    </section>
  )
}
