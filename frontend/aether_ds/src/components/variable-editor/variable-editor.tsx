import { DownloadSimple, Eye, EyeSlash, Plus, Trash, UploadSimple } from '@phosphor-icons/react'
import { Button } from '../button/button'
import { IconButton } from '../icon-button/icon-button'
import { Tooltip } from '../tooltip/tooltip'
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
  className?: string
}

function createVariableID() {
  return `variable-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function VariableEditor({
  onChange,
  onExport,
  onImport,
  onBulkEdit,
  className = '',
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
    <section className={`flex min-h-0 flex-col overflow-hidden rounded-lg border border-border ${className}`}>
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-border p-3">
        <span className="text-body-sm font-semibold">Variables</span>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            icon={UploadSimple}
            onClick={() => {
              const imported = onImport?.()
              if (imported) update(imported)
            }}
          >
            Import
          </Button>
          {onBulkEdit ? (
            <button
              type="button"
              onClick={() => onBulkEdit(variables)}
              className="text-body-sm text-muted-foreground hover:text-foreground"
            >
              Bulk edit
            </button>
          ) : null}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            icon={DownloadSimple}
            onClick={onExport}
            disabled={!onExport}
          >
            Export
          </Button>
        </div>
      </header>
      <div className="min-h-0 flex-1 divide-y divide-border overflow-y-auto">
        {variables.map((variable, index) => (
          <div
            key={variable.id}
            className="grid min-w-0 items-start gap-2 p-3"
            style={{ gridTemplateColumns: 'minmax(0, 1.5fr) minmax(0, 1fr) auto' }}
          >
            <div className="min-w-0 space-y-1" style={{ gridColumn: 1 }}>
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
                className="h-9 w-full rounded-md border border-border bg-surface-control px-2 hover:bg-surface-container-highest/40 font-mono text-code-md outline-none transition-colors placeholder:text-muted-foreground focus:border-primary focus-visible:ring-2 focus-visible:ring-ring"
              />
              {duplicates.has(variable.id) ? (
                <span className="text-label-caps text-status-danger">Duplicate key</span>
              ) : null}
            </div>
            <div className="relative min-w-0" style={{ gridColumn: 2 }}>
              <input
                aria-label="Variable value"
                type={revealed.includes(variable.id) ? 'text' : 'password'}
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
                className="h-9 w-full rounded-md border border-border bg-surface-control px-2 hover:bg-surface-container-highest/40 pr-10 font-mono text-code-md outline-none transition-colors placeholder:text-muted-foreground focus:border-primary focus-visible:ring-2 focus-visible:ring-ring"
              />
              <IconButton
                type="button"
                label={revealed.includes(variable.id) ? "Hide value" : "Show value"}
                title={revealed.includes(variable.id) ? "Hide value" : "Show value"}
                size="sm"
                icon={revealed.includes(variable.id) ? EyeSlash : Eye}
                onClick={() =>
                  setRevealed(
                    revealed.includes(variable.id)
                      ? revealed.filter((id) => id !== variable.id)
                      : [...revealed, variable.id],
                  )
                }
                className="absolute inset-y-0 right-2 my-auto"
              />
            </div>
            <Tooltip content="Remove variable">
              <IconButton
                type="button"
                label="Remove variable"
                size="sm"
                icon={Trash}
                onClick={() =>
                  update(variables.filter((item) => item.id !== variable.id))
                }
                className="self-center hover:bg-action-danger/10 hover:text-status-danger"
                style={{ gridColumn: 3, width: '2.25rem', height: '2.25rem', alignSelf: 'center' }}
              />
            </Tooltip>
          </div>
        ))}
      </div>
      <div className="border-t border-border p-3">
        <button
          type="button"
          onClick={() =>
            update([
              ...variables,
              { id: createVariableID(), key: '', value: '' },
            ])
          }
          className="inline-flex h-9 w-full items-center justify-center gap-1 rounded-md border border-border text-body-sm text-muted-foreground transition-colors hover:bg-surface-container hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:bg-surface-container-high"
        >
          <Plus size={16} />
          Add variable
        </button>
      </div>
    </section>
  )
}
