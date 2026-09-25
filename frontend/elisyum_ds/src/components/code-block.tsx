import { CopyButton } from './copy-button'

export interface CodeBlockProps {
  code: string
  language?: string
  className?: string
}

export function CodeBlock({ className = '', code, language }: CodeBlockProps) {
  return (
    <div
      className={`relative overflow-hidden rounded-lg border border-border-subtle bg-code-canvas ${className}`}
    >
      <div className="flex items-center justify-between border-b border-border-subtle px-3 py-2 text-log text-text-subtle">
        <span>{language ?? 'text'}</span>
        <CopyButton
          label={`Copy ${language ?? 'text'} code`}
          value={code}
        />
      </div>
      <pre
        aria-label={`${language ?? 'text'} code`}
        className="cursor-text overflow-auto p-4 font-technical text-code text-text-secondary"
      >
        <code>{code}</code>
      </pre>
    </div>
  )
}
