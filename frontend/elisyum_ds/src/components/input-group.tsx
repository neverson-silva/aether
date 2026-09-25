import type { ReactNode } from 'react'

export interface InputGroupProps {
  children: ReactNode
  leading?: ReactNode
  trailing?: ReactNode
  className?: string
}

export function InputGroup({
  children,
  className = '',
  leading,
  trailing,
}: InputGroupProps) {
  return (
    <div
      className={`flex min-h-10 w-full items-center rounded-md border border-border-default bg-field px-3 text-text-primary transition-[background-color,border-color,box-shadow] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-field-hover focus-within:border-focus focus-within:ring-2 focus-within:ring-focus/25 has-[[aria-invalid=true]]:border-danger has-[[aria-invalid=true]]:focus-within:border-danger has-[[aria-invalid=true]]:focus-within:ring-danger/25 ${className}`}
    >
      {leading ? (
        <span className="flex shrink-0 items-center text-text-tertiary">{leading}</span>
      ) : null}
      <div className="min-w-0 flex-1 [&>input]:min-h-9 [&>input]:border-0 [&>input]:bg-transparent [&>input]:px-2 [&>input]:ring-0 [&>input]:focus:ring-0">
        {children}
      </div>
      {trailing ? (
        <span className="flex shrink-0 items-center text-text-tertiary">
          {trailing}
        </span>
      ) : null}
    </div>
  )
}
