import { Button, Checkbox, Input } from '@aether/elisyum-ds'
import { Eye, EyeSlash, Plus, X } from '@phosphor-icons/react'

export type EnvironmentVariableDraft = {
  id: string
  name: string
  value: string
  secret: boolean
  revealed: boolean
}

export function EnvironmentVariableEditor({
  variables,
  onChange,
}: {
  variables: EnvironmentVariableDraft[]
  onChange: (variables: EnvironmentVariableDraft[]) => void
}) {
  const addVariable = () =>
    onChange([
      ...variables,
      {
        id: crypto.randomUUID(),
        name: '',
        value: '',
        secret: false,
        revealed: true,
      },
    ])

  const updateVariable = (id: string, change: Partial<EnvironmentVariableDraft>) =>
    onChange(
      variables.map((variable) =>
        variable.id === id ? { ...variable, ...change } : variable,
      ),
    )

  return (
    <section className="grid gap-3 border-t border-border-subtle pt-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="grid gap-1">
          <h3 className="text-label font-semibold text-text-primary">
            Environment variables
          </h3>
          <p className="text-supporting text-text-tertiary">
            Configure values available to the service at runtime.
          </p>
        </div>
        <Button
          type="button"
          tone="neutral"
          onClick={addVariable}
        >
          <Plus size={16} />
          Add variable
        </Button>
      </div>
      {variables.map((variable) => {
        const invalidName =
          variable.name.length > 0 &&
          !/^[A-Za-z_][A-Za-z0-9_]*$/.test(variable.name.trim())
        const duplicate =
          variable.name.trim().length > 0 &&
          variables.filter(
            (candidate) => candidate.name.trim() === variable.name.trim(),
          ).length > 1
        return (
          <div
            key={variable.id}
            className="grid min-w-0 gap-3 rounded-lg border border-border-subtle bg-surface-1 p-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_auto_auto] sm:items-center"
          >
            <Input
              aria-label="Variable name"
              aria-invalid={invalidName || duplicate}
              autoComplete="off"
              placeholder="VARIABLE_NAME"
              value={variable.name}
              onChange={(event) =>
                updateVariable(variable.id, { name: event.target.value })
              }
            />
            <div className="flex min-w-0 items-center gap-2">
              <Input
                aria-label={`Value for ${variable.name || 'new variable'}`}
                autoComplete="new-password"
                type={variable.secret && !variable.revealed ? 'password' : 'text'}
                value={variable.value}
                onChange={(event) =>
                  updateVariable(variable.id, { value: event.target.value })
                }
              />
              {variable.secret ? (
                <button
                  type="button"
                  aria-label={
                    variable.revealed ? 'Mask secret value' : 'Reveal secret value'
                  }
                  className="grid size-9 shrink-0 place-items-center rounded-md text-text-tertiary hover:bg-surface-2 hover:text-text-primary focus-visible:outline-2 focus-visible:outline-focus"
                  onClick={() =>
                    updateVariable(variable.id, { revealed: !variable.revealed })
                  }
                >
                  {variable.revealed ? <EyeSlash size={17} /> : <Eye size={17} />}
                </button>
              ) : null}
            </div>
            <label
              htmlFor={`${variable.id}-secret`}
              className="flex items-center gap-2 text-supporting text-text-secondary"
            >
              <Checkbox
                id={`${variable.id}-secret`}
                checked={variable.secret}
                onChange={(event) =>
                  updateVariable(variable.id, {
                    secret: event.target.checked,
                    revealed: true,
                  })
                }
              />
              Secret
            </label>
            <button
              type="button"
              aria-label={`Remove ${variable.name || 'environment variable'}`}
              className="grid size-9 place-items-center rounded-md text-text-tertiary hover:bg-danger-soft hover:text-danger focus-visible:outline-2 focus-visible:outline-focus"
              onClick={() =>
                onChange(variables.filter((candidate) => candidate.id !== variable.id))
              }
            >
              <X size={17} />
            </button>
            {invalidName || duplicate ? (
              <span className="text-label text-danger sm:col-span-4">
                {duplicate
                  ? 'Variable names must be unique.'
                  : 'Use letters, numbers, and underscores; the first character must be a letter or underscore.'}
              </span>
            ) : null}
          </div>
        )
      })}
    </section>
  )
}
