import { forwardRef, type InputHTMLAttributes } from 'react'

export type SliderProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>

export const Slider = forwardRef<HTMLInputElement, SliderProps>(function SliderView(
  { className = '', ...props },
  ref,
) {
  return (
    <input
      {...props}
      ref={ref}
      type="range"
      className={`h-2 w-full cursor-pointer accent-action transition-[filter,opacity] duration-[var(--ely-duration-fast)] hover:brightness-110 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-focus motion-reduce:transition-none aria-invalid:outline-2 aria-invalid:outline-offset-4 aria-invalid:outline-danger disabled:cursor-not-allowed disabled:opacity-45 ${className}`}
    />
  )
})
