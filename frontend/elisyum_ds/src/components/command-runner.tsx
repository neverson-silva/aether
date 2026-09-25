import type { ReactNode } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'
import { Button } from './button'
import { InputGroup } from './input-group'
import { Input } from './input'
import { Spinner } from './spinner'

export interface CommandRunnerProps {
  inputProps?: UseFormRegisterReturn
  value?: string
  onValueChange?: (value: string) => void
  onRun?: () => void
  running?: boolean
  result?: ReactNode
  className?: string
}
export function CommandRunner({
  className = '',
  inputProps,
  onRun,
  onValueChange,
  result,
  running = false,
  value = '',
}: CommandRunnerProps) {
  return (
    <div
      aria-busy={running}
      className={`grid gap-3 ${className}`}
    >
      <form
        className="flex flex-wrap gap-2 border-y border-border-subtle bg-surface-0 py-3 sm:flex-nowrap"
        onSubmit={(event) => {
          event.preventDefault()
          onRun?.()
        }}
      >
        <span
          aria-hidden="true"
          className="self-center pl-2 font-technical text-log text-success"
        >
          $
        </span>
        <InputGroup className="min-w-0 flex-1">
          <Input
            aria-label="Command"
            {...inputProps}
            onChange={(event) => {
              inputProps?.onChange?.(event)
              onValueChange?.(event.target.value)
            }}
            value={value}
          />
        </InputGroup>
        <Button
          disabled={running}
          type="submit"
        >
          {running ? (
            <>
              <Spinner
                aria-hidden="true"
                size="sm"
              />
              <span>Running</span>
            </>
          ) : (
            'Run command'
          )}
        </Button>
      </form>
      {result ? (
        <section
          aria-label="Command result"
          className="grid gap-2 border-y border-border-subtle bg-code-canvas p-3"
        >
          <span className="font-technical text-log text-text-subtle">Output</span>
          <pre className="cursor-text overflow-auto whitespace-pre-wrap font-technical text-log text-text-secondary">
            {result}
          </pre>
        </section>
      ) : (
        <p className="border-y border-border-subtle px-3 py-5 text-supporting text-text-tertiary">
          Enter a command to inspect its output.
        </p>
      )}
    </div>
  )
}
