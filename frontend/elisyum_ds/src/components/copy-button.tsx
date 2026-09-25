import { Check, Copy, WarningCircle } from '@phosphor-icons/react'
import { useState } from 'react'
import { IconButton } from './icon-button'

export interface CopyButtonProps {
  value: string
  label?: string
  className?: string
}

export function CopyButton({
  className = '',
  label = 'Copy value',
  value,
}: CopyButtonProps) {
  const [status, setStatus] = useState<'idle' | 'copied' | 'failed'>('idle')
  const copy = async () => {
    try {
      if (!navigator.clipboard) throw new Error('Clipboard unavailable')
      await navigator.clipboard.writeText(value)
      setStatus('copied')
      window.setTimeout(() => setStatus('idle'), 1400)
    } catch {
      setStatus('failed')
      window.setTimeout(() => setStatus('idle'), 1800)
    }
  }
  const icon =
    status === 'copied' ? (
      <Check
        aria-hidden="true"
        size={15}
      />
    ) : status === 'failed' ? (
      <WarningCircle
        aria-hidden="true"
        size={15}
      />
    ) : (
      <Copy
        aria-hidden="true"
        size={15}
      />
    )
  return (
    <>
      <IconButton
        className={className}
        label={
          status === 'copied' ? 'Copied' : status === 'failed' ? 'Copy failed' : label
        }
        onClick={copy}
        size="sm"
      >
        {icon}
      </IconButton>
      <span
        aria-live="polite"
        className="sr-only"
      >
        {status === 'copied' ? 'Copied' : status === 'failed' ? 'Copy failed' : ''}
      </span>
    </>
  )
}
