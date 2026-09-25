import { forwardRef, type TextareaHTMLAttributes } from 'react'
import { inputControlClasses } from './input-styles'

export interface CodeEditorLiteProps
  extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  language?: string
}

export const CodeEditorLite = forwardRef<HTMLTextAreaElement, CodeEditorLiteProps>(
  function CodeEditorLiteView({ className = '', language, ...props }, ref) {
    const invalid = (props as { 'data-invalid'?: string })['data-invalid']
    return (
      <div
        className="grid overflow-hidden rounded-md border border-border-subtle bg-code-canvas data-[invalid=true]:border-danger"
        data-invalid={invalid}
      >
        <div className="flex items-center justify-between border-b border-border-subtle px-3 py-2 font-technical text-log text-text-subtle">
          <span>{language ?? 'text'}</span>
          <span className="text-text-tertiary">Editable</span>
        </div>
        <textarea
          {...props}
          ref={ref}
          spellCheck={false}
          className={`${inputControlClasses} min-h-48 resize-y rounded-none border-0 bg-field p-4 font-technical text-code text-text-secondary focus:ring-2 focus:ring-focus/25 ${className}`}
        />
      </div>
    )
  },
)
