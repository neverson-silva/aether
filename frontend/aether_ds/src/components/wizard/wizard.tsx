import { Check } from "@phosphor-icons/react";
import { type ReactNode, useState } from "react";
export interface WizardStep {
  id: string;
  title: string;
  description?: string;
  content: ReactNode;
  validate?: () => boolean | Promise<boolean>;
}
export interface WizardProps {
  steps: WizardStep[];
  initialStep?: number;
  currentStep?: number;
  onStepChange?: (step: number) => void;
  onNext?: (currentStep: number, nextStep: number) => void | Promise<void>;
  onComplete?: () => void;
  onCancel?: () => void;
  loading?: boolean;
  children?: ReactNode;
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
  const [internalCurrent, setInternalCurrent] = useState(initialStep);
  const current = currentStep ?? internalCurrent;
  const [invalid, setInvalid] = useState(false);
  const [advancing, setAdvancing] = useState(false);
  const step = steps[current];
  const setCurrent = (next: number) => {
    setInternalCurrent(next);
    onStepChange?.(next);
  };
  const next = async () => {
    if (advancing) return;
    setAdvancing(true);
    const valid = (await step.validate?.()) ?? true;
    if (!valid) {
      setInvalid(true);
      setAdvancing(false);
      return;
    }
    setInvalid(false);
    try {
      if (current === steps.length - 1) {
        onComplete?.();
        return;
      }
      await onNext?.(current, current + 1);
      setCurrent(current + 1);
    } finally {
      setAdvancing(false);
    }
  };
  return (
    <section className="flex h-full max-h-[calc(100vh-2rem)] min-h-0 flex-col overflow-hidden rounded-2xl border border-border bg-surface-card shadow-md">
      <nav
        aria-label="Wizard progress"
        className="flex shrink-0 overflow-x-auto border-b border-border bg-surface-container/55 p-4 backdrop-blur-xl"
      >
        <ol className="flex min-w-max items-center gap-3">
          {steps.map((item, index) => (
            <li key={item.id} className="flex items-center gap-3">
              <button
                type="button"
                disabled={index > current}
                onClick={() => index <= current && setCurrent(index)}
                aria-current={index === current ? "step" : undefined}
                className={`flex items-center gap-2 rounded-lg px-2 py-1 text-body-sm outline-none transition-[background-color,color,transform] duration-150 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] ${index === current ? "bg-surface-card font-semibold text-primary shadow-sm" : index < current ? "text-foreground hover:bg-surface-card/70" : "text-muted-foreground"}`}
              >
                <span
                  className={`inline-flex size-8 items-center justify-center rounded-full border text-body-sm font-semibold transition-colors ${index < current ? "border-status-success bg-status-success text-status-success-foreground" : index === current ? "border-primary bg-primary text-primary-foreground shadow-sm" : "border-border bg-surface-card"}`}
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
      <div className="min-h-0 flex-1 overflow-y-auto p-6 sm:p-8">
        <div className="mb-6">
          <h2 className="font-display-md text-headline-sm tracking-[-0.025em] text-foreground">
            {step.title}
          </h2>
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
      <footer className="flex shrink-0 flex-wrap justify-between gap-3 border-t border-border bg-surface-container/35 p-4 backdrop-blur-xl">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg px-3 py-2 text-body-sm text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
        >
          Cancel
        </button>
        <div className="flex gap-2">
          {current > 0 ? (
            <button
              type="button"
              onClick={() => setCurrent(current - 1)}
              className="rounded-lg border border-border px-3 py-2 text-body-sm outline-none transition-[background-color,border-color,transform] duration-150 hover:border-primary/50 hover:bg-surface-container focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
            >
              Back
            </button>
          ) : null}
          <button
            type="button"
            disabled={loading || advancing}
            onClick={next}
            className="rounded-lg bg-primary px-4 py-2 text-body-sm font-semibold text-primary-foreground shadow-sm outline-none transition-[background-color,box-shadow,transform] duration-150 hover:bg-primary/90 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985] disabled:pointer-events-none disabled:opacity-50"
          >
            {loading
              ? "Saving..."
              : current === steps.length - 1
                ? "Finish"
                : "Next"}
          </button>
        </div>
      </footer>
    </section>
  );
}
