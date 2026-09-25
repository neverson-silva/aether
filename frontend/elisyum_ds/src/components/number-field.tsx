import { forwardRef, type InputHTMLAttributes } from 'react'
import { inputControlClasses } from './input-styles'

export type NumberFieldProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>

export const NumberField = forwardRef<HTMLInputElement, NumberFieldProps>(
  function NumberFieldView({ className = '', ...props }, ref) {
    return (
      <input
        {...props}
        ref={ref}
        type="number"
        className={`${inputControlClasses} ${className}`}
      />
    )
  },
)
