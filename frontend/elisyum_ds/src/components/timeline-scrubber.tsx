import { forwardRef, type InputHTMLAttributes } from 'react'

export type TimelineScrubberProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  'type'
> & { label?: string }

export const TimelineScrubber = forwardRef<HTMLInputElement, TimelineScrubberProps>(
  function TimelineScrubberView(
    { className = '', label = 'Timeline position', ...props },
    ref,
  ) {
    return (
      <label className="grid gap-2">
        <span className="text-label text-text-secondary">{label}</span>
        <input
          {...props}
          ref={ref}
          aria-label={label}
          type="range"
          className={`h-2 w-full cursor-pointer accent-action transition-[filter,opacity] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:brightness-110 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-45 ${className}`}
        />
      </label>
    )
  },
)
