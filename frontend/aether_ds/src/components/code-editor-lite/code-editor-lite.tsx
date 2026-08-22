import { Copy, FileCode } from '@phosphor-icons/react'
import { useMemo, useState } from 'react'
import { Button } from '../button/button'

export interface CodeEditorLiteProps {
  value?: string
  defaultValue?: string
  language?: string
  filename?: string
  readOnly?: boolean
  lineNumbers?: boolean
  onValueChange?: (value: string) => void
  className?: string
}
export function CodeEditorLite({
  className = '',
  defaultValue = '',
  filename,
  language = 'text',
  lineNumbers = true,
  onValueChange,
  readOnly = false,
  value,
}: CodeEditorLiteProps) {
  const [internalValue, setInternalValue] = useState(defaultValue)
  const [copied, setCopied] = useState(false)
  const currentValue = value === undefined ? internalValue : value
  const lines = useMemo(() => currentValue.split('\n'), [currentValue])
  const update = (next: string) => {
    setInternalValue(next)
    onValueChange?.(next)
  }
  const copy = async () => {
    await navigator.clipboard?.writeText(currentValue)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1200)
  }
  return (
    <div
      className={`overflow-hidden rounded-lg border border-border bg-surface-lowest ${className}`}
    >
      <div className="flex items-center gap-2 border-b border-border bg-surface-container px-3 py-2">
        <FileCode size={16} className="text-primary" aria-hidden="true" />
        <span className="min-w-0 flex-1 truncate text-body-sm text-foreground">
          {filename ?? language}
        </span>
        <Button
          size="sm"
          variant="quiet"
          icon={Copy}
          aria-label="Copy code"
          onClick={copy}
        >
          {copied ? 'Copied' : 'Copy'}
        </Button>
      </div>
      <div className="flex max-h-[32rem] min-h-40 overflow-auto font-code-md text-code-md">
        <div
          className="select-none border-r border-border bg-surface-container-low px-3 py-3 text-right text-muted-foreground"
          aria-hidden="true"
        >
          {lineNumbers
            ? lines.map((_, index) => <div key={index}>{index + 1}</div>)
            : null}
        </div>
        <textarea
          aria-label={filename ?? `${language} editor`}
          readOnly={readOnly}
          value={currentValue}
          onChange={(event) => update(event.target.value)}
          spellCheck={false}
          className="min-h-40 min-w-[36rem] flex-1 resize-none bg-transparent px-3 py-3 text-foreground outline-none placeholder:text-muted-foreground"
        />
      </div>
    </div>
  )
}
