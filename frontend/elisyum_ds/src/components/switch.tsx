import { forwardRef, type InputHTMLAttributes } from 'react'

export type SwitchProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>

export const Switch = forwardRef<HTMLInputElement, SwitchProps>(function SwitchView(
  { className = '', ...props },
  ref,
) {
  return (
    <label
      className={`inline-flex min-h-7 cursor-pointer items-center has-[:disabled]:cursor-not-allowed ${className}`}
    >
      <input
        {...props}
        ref={ref}
        type="checkbox"
        className="peer sr-only"
      />
      <span
        aria-hidden="true"
        className="relative h-5 w-9 rounded-pill bg-surface-4 transition-[background-color,box-shadow,transform] duration-[var(--ely-duration-standard)] ease-ely-out peer-checked:bg-action-strong peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-focus peer-aria-invalid:outline-2 peer-aria-invalid:outline-offset-2 peer-aria-invalid:outline-danger peer-disabled:opacity-45 after:absolute after:left-1 after:top-1 after:size-3 after:rounded-full after:bg-text-primary after:shadow-elevation-1 after:transition-[background-color,box-shadow,transform] after:duration-[var(--ely-duration-standard)] after:ease-ely-out after:will-change-transform after:content-[''] peer-active:after:scale-[.92] peer-checked:after:translate-x-4 peer-checked:after:bg-on-action peer-checked:after:shadow-elevation-2 motion-reduce:transition-none motion-reduce:after:transition-none motion-reduce:peer-active:after:scale-100"
      />
    </label>
  )
})
