import {
  useEffect,
  useMemo,
  useState,
  type ClipboardEvent,
  type KeyboardEvent,
  type ReactNode,
} from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'

export interface InputOTPProps {
  length?: number
  value?: string
  defaultValue?: string
  onChange?: (value: string) => void
  inputProps?: UseFormRegisterReturn
  label?: ReactNode
  disabled?: boolean
  className?: string
}

export function InputOTP({
  className = '',
  defaultValue = '',
  disabled = false,
  inputProps,
  label = 'One-time passcode',
  length = 6,
  onChange,
  value: controlledValue,
}: InputOTPProps) {
  const [uncontrolledValue, setUncontrolledValue] = useState(defaultValue)
  const value = controlledValue ?? uncontrolledValue
  const digits = useMemo(
    () => Array.from({ length }, (_, index) => value[index] ?? ''),
    [length, value],
  )
  const slotIds = useMemo(
    () => Array.from({ length }, (_, index) => `${inputProps?.name ?? 'otp'}-${index}`),
    [inputProps?.name, length],
  )
  const invalid = Boolean(
    (
      inputProps as
        | { 'aria-invalid'?: boolean | string; 'data-invalid'?: string }
        | undefined
    )?.['aria-invalid'] ||
      (inputProps as { 'data-invalid'?: string } | undefined)?.['data-invalid'],
  )

  useEffect(() => {
    if (controlledValue !== undefined) setUncontrolledValue(controlledValue)
  }, [controlledValue])

  const updateValue = (nextValue: string) => {
    const next = nextValue.replace(/\D/g, '').slice(0, length)
    if (controlledValue === undefined) setUncontrolledValue(next)
    onChange?.(next)
    if (inputProps?.onChange)
      inputProps.onChange({
        target: { name: inputProps.name, value: next, type: 'text' },
      } as unknown as Event)
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>, index: number) => {
    if (event.key === 'Backspace' && !digits[index] && index > 0)
      document.getElementById(`${inputProps?.name ?? 'otp'}-${index - 1}`)?.focus()
    if (event.key === 'ArrowLeft' && index > 0)
      document.getElementById(`${inputProps?.name ?? 'otp'}-${index - 1}`)?.focus()
    if (event.key === 'ArrowRight' && index < length - 1)
      document.getElementById(`${inputProps?.name ?? 'otp'}-${index + 1}`)?.focus()
  }
  const handlePaste = (event: ClipboardEvent<HTMLInputElement>) => {
    event.preventDefault()
    const pasted = event.clipboardData
      .getData('text')
      .replace(/\D/g, '')
      .slice(0, length)
    if (!pasted) return
    updateValue(pasted)
    requestAnimationFrame(() =>
      document
        .getElementById(
          `${inputProps?.name ?? 'otp'}-${Math.min(pasted.length, length - 1)}`,
        )
        ?.focus(),
    )
  }

  return (
    <div className={`grid gap-2 ${className}`}>
      <span className="text-label text-text-secondary">{label}</span>
      <input
        {...inputProps}
        aria-hidden="true"
        tabIndex={-1}
        value={value}
        onChange={() => undefined}
        className="sr-only"
      />
      <fieldset
        aria-label={String(label)}
        className="flex gap-2"
      >
        {digits.map((digit, index) => (
          <input
            key={slotIds[index]}
            id={`${inputProps?.name ?? 'otp'}-${index}`}
            aria-invalid={invalid || undefined}
            aria-label={`${label} digit ${index + 1} of ${length}`}
            autoComplete={index === 0 ? 'one-time-code' : 'off'}
            disabled={disabled}
            inputMode="numeric"
            maxLength={1}
            type="text"
            value={digit}
            onChange={(event) => {
              const next = [...digits]
              next[index] = event.target.value.slice(-1)
              updateValue(next.join(''))
              if (event.target.value && index < length - 1)
                document
                  .getElementById(`${inputProps?.name ?? 'otp'}-${index + 1}`)
                  ?.focus()
            }}
            onKeyDown={(event) => handleKeyDown(event, index)}
            onPaste={handlePaste}
            className={`size-11 cursor-text rounded-md border bg-field text-center text-body text-text-primary outline-none transition-[background-color,border-color,box-shadow,transform] duration-[var(--ely-duration-fast)] hover:bg-field-hover focus:border-focus focus:ring-2 focus:ring-focus/25 motion-safe:active:scale-[var(--ely-motion-press-scale)] disabled:cursor-not-allowed disabled:opacity-45 ${invalid ? 'border-danger focus:border-danger focus:ring-danger/25' : 'border-border-default'}`}
          />
        ))}
      </fieldset>
    </div>
  )
}
