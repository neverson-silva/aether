import type { ReactNode } from 'react'
import { RadioGroup, type RadioOption } from './radio-group'

export interface QuestionnaireQuestion {
  id: string
  prompt: ReactNode
  options: RadioOption[]
}
export interface QuestionnaireProps {
  questions: QuestionnaireQuestion[]
  values?: Record<string, string>
  onValueChange?: (id: string, value: string) => void
  className?: string
}

export function Questionnaire({
  className = '',
  onValueChange,
  questions,
  values = {},
}: QuestionnaireProps) {
  return (
    <div className={`grid gap-6 ${className}`}>
      {questions.map((question, index) => (
        <fieldset
          className="grid gap-3 rounded-lg border border-border-subtle bg-surface-1 px-4 py-4"
          key={question.id}
        >
          <legend className="text-overline text-action">
            Question {String(index + 1).padStart(2, '0')}
          </legend>
          <p className="text-body text-text-primary">{question.prompt}</p>
          <RadioGroup
            aria-label={String(question.prompt)}
            onValueChange={(value) => onValueChange?.(question.id, value)}
            options={question.options}
            value={values[question.id]}
          />
        </fieldset>
      ))}
    </div>
  )
}
