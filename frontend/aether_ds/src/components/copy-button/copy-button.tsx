import { Check, Copy } from '@phosphor-icons/react'
import { useState } from 'react'
export interface CopyButtonProps {
  value: string
  label?: string
  onCopy?: () => void
  sensitive?: boolean
}
export function CopyButton({
  label = 'Copy',
  onCopy,
  sensitive,
  value,
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      onCopy?.()
      window.setTimeout(() => setCopied(false), 1400)
    } catch {
      setCopied(false)
    }
  }
  return (
    <button
      type="button"
      onClick={copy}
      aria-label={`${label}${sensitive ? ' sensitive value' : ''}`}
      className="inline-flex items-center gap-1.5 rounded-md border border-border px-2 py-1.5 text-body-sm text-muted-foreground transition-colors hover:bg-surface-container hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
    >
      {copied ? (
        <Check size={16} className="text-status-success" />
      ) : (
        <Copy size={16} />
      )}
      {copied ? 'Copied' : label}
    </button>
  )
}
