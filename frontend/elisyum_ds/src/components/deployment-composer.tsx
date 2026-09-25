import type { FieldValues, UseFormRegister } from 'react-hook-form'
import { Button } from './button'
import { Field } from './field'
import { Select } from './select'
import { Textarea } from './textarea'

export interface DeploymentComposerProps<T extends FieldValues> {
  register: UseFormRegister<T>
  onSubmit?: () => void
  className?: string
}
export function DeploymentComposer<T extends FieldValues>({
  className = '',
  onSubmit,
  register,
}: DeploymentComposerProps<T>) {
  return (
    <form
      aria-label="Deployment configuration"
      className={`grid gap-6 rounded-xl border border-border-subtle bg-surface-1 p-5 ${className}`}
      onSubmit={(event) => {
        event.preventDefault()
        onSubmit?.()
      }}
    >
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border-subtle pb-4">
        <div>
          <p className="text-overline text-action">Release workflow</p>
          <h2 className="text-section-title text-text-primary">Prepare deployment</h2>
          <p className="mt-1 max-w-xl text-supporting text-text-secondary">
            Choose the target and leave an operational note before reviewing the
            release.
          </p>
        </div>
        <span className="rounded-sm border border-border-subtle bg-surface-2 px-2 py-1 font-mono text-log text-text-tertiary">
          STEP 01 / 02
        </span>
      </div>
      <div className="grid gap-5 sm:grid-cols-[minmax(0,0.65fr)_minmax(0,1.35fr)] sm:items-start">
        <Field
          id="deployment-environment"
          label="Environment"
        >
          <Select
            id="deployment-environment"
            {...register('environment' as never)}
          >
            <option value="production">Production</option>
            <option value="staging">Staging</option>
          </Select>
        </Field>
        <Field
          id="deployment-notes"
          label="Release notes"
          description="Describe the change for the deployment record."
        >
          <Textarea
            id="deployment-notes"
            {...register('notes' as never)}
            placeholder="Describe this deployment"
          />
        </Field>
      </div>
      <div className="flex items-center justify-between gap-4 border-t border-border-subtle pt-4">
        <p className="text-log text-text-tertiary">
          Review includes environment, notes and current service state.
        </p>
        <Button type="submit">Review deployment</Button>
      </div>
    </form>
  )
}
