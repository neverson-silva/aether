import { Check } from '@phosphor-icons/react'
import { type ReactNode, useState } from 'react'
export interface WizardStep {
  id: string
  title: string
  description?: string
  content: ReactNode
  validate?: () => boolean | Promise<boolean>
}
export interface WizardProps {
  steps: WizardStep[]
  initialStep?: number
  currentStep?: number
  onStepChange?: (step: number) => void
  onNext?: (currentStep: number, nextStep: number) => void | Promise<void>
  onComplete?: () => void
  onCancel?: () => void
  loading?: boolean
  children?: ReactNode
}
export function Wizard({
  children,
  currentStep,
  initialStep = 0,
  loading,
  onStepChange,
  onNext,
  onCancel,
  onComplete,
  steps,
}: WizardProps) {
  const [internalCurrent, setInternalCurrent] = useState(initialStep)
  const current = currentStep ?? internalCurrent
  const [invalid, setInvalid] = useState(false)
  const [advancing, setAdvancing] = useState(false)
  const step = steps[current]
  const setCurrent = (next: number) => {
    setInternalCurrent(next)
    onStepChange?.(next)
  }
  const next = async () => {
    if (advancing) return
    setAdvancing(true)
    const valid = (await step.validate?.()) ?? true
    if (!valid) {
      setInvalid(true)
      setAdvancing(false)
      return
    }
    setInvalid(false)
    try {
      if (current === steps.length - 1) {
        onComplete?.()
        return
      }
      await onNext?.(current, current + 1)
      setCurrent(current + 1)
    } finally {
      setAdvancing(false)
    }
  }
  return (
    <section className="flex h-full max-h-[calc(100vh-2rem)] min-h-0 flex-col overflow-hidden rounded-xl border border-border bg-surface-card">
      <nav
        aria-label="Wizard progress"
        className="flex shrink-0 overflow-x-auto border-b border-border p-4"
      >
        <ol className="flex min-w-max items-center gap-3">
          {steps.map((item, index) => (
            <li key={item.id} className="flex items-center gap-3">
              <button
                type="button"
                disabled={index > current}
                onClick={() => index <= current && setCurrent(index)}
                className={`flex items-center gap-2 text-body-sm ${index === current ? 'font-semibold text-primary' : index < current ? 'text-foreground' : 'text-muted-foreground'}`}
              >
                <span
                  className={`inline-flex size-8 items-center justify-center rounded-full border text-body-sm font-semibold ${index < current ? 'border-status-success bg-status-success text-status-success-foreground' : index === current ? 'border-primary bg-primary text-primary-foreground' : 'border-border'}`}
                >
                  {index < current ? <Check size={16} /> : index + 1}
                </span>
                {item.title}
              </button>
              {index < steps.length - 1 ? (
                <span className="h-px w-8 bg-border" />
              ) : null}
            </li>
          ))}
        </ol>
      </nav>
      <div className="min-h-0 flex-1 overflow-y-auto p-6">
        <div className="mb-6">
          <h2 className="text-headline-sm text-foreground">{step.title}</h2>
          {step.description ? (
            <p className="mt-1 text-body-sm text-muted-foreground">
              {step.description}
            </p>
          ) : null}
        </div>
        {children ?? step.content}
        {invalid ? (
          <p role="alert" className="mt-4 text-body-sm text-status-danger">
            Review the required fields before continuing.
          </p>
        ) : null}
      </div>
      <footer className="flex shrink-0 flex-wrap justify-between gap-3 border-t border-border p-4">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-md px-3 py-2 text-body-sm text-muted-foreground hover:bg-surface-container"
        >
          Cancel
        </button>
        <div className="flex gap-2">
          {current > 0 ? (
            <button
              type="button"
              onClick={() => setCurrent(current - 1)}
              className="rounded-md border border-border px-3 py-2 text-body-sm"
            >
              Back
            </button>
          ) : null}
            <button
              type="button"
              disabled={loading || advancing}
            onClick={next}
            className="rounded-md bg-primary px-3 py-2 text-body-sm text-primary-foreground disabled:opacity-50"
          >
            {loading
              ? 'Saving...'
              : current === steps.length - 1
                ? 'Finish'
                : 'Next'}
          </button>
        </div>
      </footer>
    </section>
  )
}
