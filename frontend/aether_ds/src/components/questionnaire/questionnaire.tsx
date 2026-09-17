import { type ReactNode, useEffect, useState } from "react";
export interface QuestionnaireQuestion {
  id: string;
  title: ReactNode;
  description?: ReactNode;
  options: { value: string; label: string }[];
  when?: (answers: Record<string, string>) => boolean;
}
export interface QuestionnaireProps {
  questions: QuestionnaireQuestion[];
  initialAnswers?: Record<string, string>;
  onChange?: (answers: Record<string, string>) => void;
  onComplete?: (answers: Record<string, string>) => void;
  autosave?: (answers: Record<string, string>) => void;
}
export function Questionnaire({
  autosave,
  initialAnswers = {},
  onChange,
  onComplete,
  questions,
}: QuestionnaireProps) {
  const [answers, setAnswers] = useState(initialAnswers);
  const visible = questions.filter(
    (question) => question.when?.(answers) ?? true,
  );
  useEffect(() => {
    onChange?.(answers);
    autosave?.(answers);
  }, [answers, autosave, onChange]);
  return (
    <div className="space-y-6">
      {visible.map((question, index) => (
        <fieldset
          key={question.id}
          className="rounded-2xl border border-border bg-surface-card p-4 shadow-sm"
        >
          <legend className="px-1 text-body-sm font-semibold text-foreground">
            {index + 1}. {question.title}
          </legend>
          {question.description ? (
            <p className="mt-1 text-body-sm text-muted-foreground">
              {question.description}
            </p>
          ) : null}
          <div className="mt-4 grid gap-2">
            {question.options.map((option) => (
              <label
                key={option.value}
                className="flex cursor-pointer items-center gap-3 rounded-xl border border-border p-3 text-body-sm outline-none transition-[background-color,border-color] duration-150 hover:border-primary/40 hover:bg-surface-container focus-within:border-primary/50 focus-within:ring-2 focus-within:ring-ring/40"
              >
                <input
                  type="radio"
                  name={question.id}
                  value={option.value}
                  checked={answers[question.id] === option.value}
                  onChange={() =>
                    setAnswers({ ...answers, [question.id]: option.value })
                  }
                  className="accent-primary"
                />
                {option.label}
              </label>
            ))}
          </div>
        </fieldset>
      ))}
      <button
        type="button"
        onClick={() => onComplete?.(answers)}
        className="rounded-xl bg-primary px-3 py-2 text-body-sm text-primary-foreground outline-none transition-[filter,transform] duration-150 hover:brightness-110 focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.985]"
      >
        Review answers
      </button>
    </div>
  );
}
