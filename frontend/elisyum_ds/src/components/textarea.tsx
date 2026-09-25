import { forwardRef, type TextareaHTMLAttributes } from 'react'
import { inputControlClasses } from './input-styles'

export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  function TextareaView({ className = '', ...props }, ref) {
    return (
      <textarea
        {...props}
        ref={ref}
        className={`${inputControlClasses} min-h-24 resize-y py-2.5 ${className}`}
      />
    )
  },
)
