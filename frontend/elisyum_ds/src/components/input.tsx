import { forwardRef, type InputHTMLAttributes } from 'react'
import { inputControlClasses } from './input-styles'

export type InputProps = InputHTMLAttributes<HTMLInputElement>

export const Input = forwardRef<HTMLInputElement, InputProps>(function InputView(
  { className = '', ...props },
  ref,
) {
  return (
    <input
      {...props}
      ref={ref}
      className={`${inputControlClasses} ${className}`}
    />
  )
})
