import { Check, CheckCircle, CaretRight, WarningCircle } from '@phosphor-icons/react'
import { useId, type ReactNode } from 'react'
import { Button } from './button'
import { Progress } from './progress'

export type WizardStepStatus = 'default' | 'complete' | 'error' | 'skipped'

export interface WizardStep {
  id: string
  label: ReactNode
  content: ReactNode
  description?: ReactNode
  summary?: ReactNode
  status?: WizardStepStatus
  canContinue?: boolean
}
export interface WizardProps {
  steps: WizardStep[]
  activeStep?: number
  onStepChange?: (step: number) => void
  onComplete?: () => void
  backLabel?: ReactNode
  continueLabel?: ReactNode
  completeLabel?: ReactNode
  className?: string
}

export function Wizard({
  activeStep = 0,
  backLabel = 'Back',
  className = '',
  completeLabel = 'Create service',
  continueLabel = 'Continue',
  onComplete,
  onStepChange,
  steps,
}: WizardProps) {
  const baseId = useId()
  if (!steps.length) return null
  const step = Math.min(Math.max(activeStep, 0), steps.length - 1)
  const current = steps[step]
  const progress = ((step + 1) / steps.length) * 100
  const isComplete =
    current.status === 'complete' || (current.status === undefined && step > 0)
  const canContinue = current.canContinue !== false
  const titleId = `${baseId}-title`
  const contentId = `${baseId}-content`
  const getStepState = (index: number, item: WizardStep) => {
    if (index === step) return item.status === 'error' ? 'error' : 'current'
    if (item.status === 'error') return 'error'
    if (item.status === 'skipped') return 'skipped'
    if (item.status === 'complete' || (item.status === undefined && index < step))
      return 'complete'
    return 'upcoming'
  }
  const changeStep = (nextStep: number) => {
    if (nextStep < 0 || nextStep >= steps.length || nextStep === step) return
    onStepChange?.(nextStep)
  }

  return (
    <section
      aria-labelledby={titleId}
      className={`grid overflow-hidden rounded-xl border border-border-default bg-surface-1 ${className}`}
    >
      <header className="border-b border-border-subtle bg-surface-2/70 px-5 py-5 sm:px-7">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="font-technical text-log text-text-tertiary">
              Create workflow
            </p>
            <h2
              className="mt-1 text-section-title text-text-primary"
              id={titleId}
            >
              {current.label}
            </h2>
            {current.description ? (
              <p className="mt-1 max-w-2xl text-body text-text-secondary">
                {current.description}
              </p>
            ) : null}
          </div>
          <div className="min-w-40 text-right">
            <p className="text-supporting text-text-tertiary">
              Step {step + 1} of {steps.length}
            </p>
            <Progress
              aria-label="Wizard progress"
              className="mt-2"
              max={100}
              value={progress}
            />
          </div>
        </div>
      </header>
      <div className="grid min-w-0 lg:grid-cols-[15rem_minmax(0,1fr)]">
        <nav
          aria-label="Setup steps"
          className="border-b border-border-subtle bg-surface-0 p-3 lg:border-b-0 lg:border-r lg:p-4"
        >
          <ol className="flex gap-2 overflow-x-auto lg:grid lg:gap-px">
            {steps.map((item, index) => {
              const state = getStepState(index, item)
              const navigable =
                index <= step || state === 'complete' || state === 'error'
              return (
                <li
                  className="min-w-44 lg:min-w-0"
                  key={item.id}
                >
                  <button
                    aria-current={state === 'current' ? 'step' : undefined}
                    aria-label={`${index + 1}. ${typeof item.label === 'string' ? item.label : `Step ${index + 1}`}${state === 'complete' ? ', complete' : state === 'error' ? ', needs attention' : ''}`}
                    className={`group flex w-full items-start gap-3 border-l-2 p-3 text-left transition-[background-color,border-color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none ${state === 'current' ? 'border-action bg-action-soft' : 'border-transparent hover:bg-surface-interactive'} ${navigable ? 'cursor-pointer' : 'cursor-not-allowed opacity-55'}`}
                    disabled={!navigable}
                    onClick={() => changeStep(index)}
                    type="button"
                  >
                    <span
                      className={`mt-0.5 grid size-7 shrink-0 place-items-center rounded-full border text-label transition-[background-color,border-color,color,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none ${state === 'current' ? 'border-action bg-action text-on-action' : state === 'complete' ? 'border-success bg-success/15 text-success-strong' : state === 'error' ? 'border-danger bg-danger/15 text-danger-strong' : 'border-border-default bg-surface-2 text-text-tertiary'} motion-reduce:group-active:scale-100 group-active:scale-[var(--ely-motion-press-scale)]`}
                    >
                      {state === 'complete' ? (
                        <Check
                          aria-hidden="true"
                          size={15}
                          weight="bold"
                        />
                      ) : state === 'error' ? (
                        <WarningCircle
                          aria-hidden="true"
                          size={16}
                          weight="bold"
                        />
                      ) : (
                        index + 1
                      )}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span
                        className={`block truncate text-label ${state === 'current' ? 'text-text-primary' : 'text-text-secondary'}`}
                      >
                        {item.label}
                      </span>
                      {item.summary ? (
                        <span className="mt-0.5 block truncate text-log text-text-tertiary">
                          {item.summary}
                        </span>
                      ) : null}
                      {state === 'complete' ? (
                        <span className="mt-0.5 block text-log text-success-strong">
                          Complete
                        </span>
                      ) : state === 'error' ? (
                        <span className="mt-0.5 block text-log text-danger-strong">
                          Needs attention
                        </span>
                      ) : null}
                    </span>
                    {state === 'current' ? (
                      <CaretRight
                        aria-hidden="true"
                        className="mt-1 shrink-0 text-action"
                        size={14}
                        weight="bold"
                      />
                    ) : null}
                  </button>
                </li>
              )
            })}
          </ol>
        </nav>
        <div className="grid min-w-0 grid-rows-[1fr_auto]">
          <div
            aria-labelledby={titleId}
            className="min-h-72 p-5 transition-[opacity,transform] duration-[var(--ely-duration-standard)] motion-reduce:transition-opacity sm:p-7"
            id={contentId}
            key={current.id}
            role="tabpanel"
            tabIndex={-1}
          >
            {current.content}
          </div>
          <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-border-subtle bg-surface-0 px-5 py-4 sm:px-7">
            <div className="flex min-w-0 items-center gap-2 text-supporting text-text-tertiary">
              {isComplete ? (
                <CheckCircle
                  aria-hidden="true"
                  className="shrink-0 text-success"
                  size={17}
                  weight="fill"
                />
              ) : null}
              <span className="truncate">
                {isComplete
                  ? 'This step is saved'
                  : 'Your progress is saved as you continue'}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Button
                disabled={step === 0}
                onClick={() => changeStep(step - 1)}
                tone="ghost"
              >
                {backLabel}
              </Button>
              {step === steps.length - 1 ? (
                <Button
                  disabled={!canContinue}
                  onClick={onComplete}
                >
                  {completeLabel}
                </Button>
              ) : (
                <Button
                  disabled={!canContinue}
                  onClick={() => changeStep(step + 1)}
                >
                  {continueLabel}
                </Button>
              )}
            </div>
          </footer>
        </div>
      </div>
    </section>
  )
}
