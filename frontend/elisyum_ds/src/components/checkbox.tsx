import { forwardRef, type InputHTMLAttributes } from 'react'

export type CheckboxProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>

export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(function CheckboxView(
  { className = '', ...props },
  ref,
) {
  return (
    <input
      {...props}
      ref={ref}
      type="checkbox"
      className={`size-4 cursor-pointer accent-action transition-transform duration-[var(--ely-duration-fast)] active:scale-95 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus motion-reduce:transition-none motion-reduce:active:scale-100 aria-invalid:outline-2 aria-invalid:outline-offset-2 aria-invalid:outline-danger disabled:cursor-not-allowed disabled:opacity-45 ${className}`}
    />
  )
})
