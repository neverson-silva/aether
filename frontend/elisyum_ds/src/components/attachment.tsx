import { FileText, X } from '@phosphor-icons/react'
import type { ReactNode } from 'react'
import { IconButton } from './icon-button'
import { Progress } from './progress'

export interface AttachmentItem {
  id: string
  name: string
  size?: string
  progress?: number
  status?: ReactNode
}
export interface AttachmentProps {
  items: AttachmentItem[]
  onRemove?: (id: string) => void
  className?: string
}

export function Attachment({ className = '', items, onRemove }: AttachmentProps) {
  return (
    <ul
      aria-label="Attachments"
      className={`m-0 grid list-none divide-y divide-border-subtle border-y border-border-subtle p-0 ${className}`}
    >
      {items.map((item) => (
        <li
          className="grid gap-2 bg-surface-1 px-3 py-3 transition-[background-color,border-color] duration-[var(--ely-duration-fast)] motion-reduce:transition-none hover:bg-surface-2"
          key={item.id}
        >
          <div className="flex items-center gap-3">
            <FileText
              aria-hidden="true"
              className="shrink-0 text-action"
              size={20}
            />
            <div className="min-w-0 flex-1">
              <p className="truncate text-supporting text-text-primary">{item.name}</p>
              <p className="text-log text-text-tertiary">
                {item.size} {item.status}
              </p>
            </div>
            {onRemove ? (
              <IconButton
                label={`Remove ${item.name}`}
                onClick={() => onRemove(item.id)}
                size="sm"
              >
                <X
                  aria-hidden="true"
                  size={16}
                />
              </IconButton>
            ) : null}
          </div>
          {item.progress !== undefined ? (
            <Progress
              label={`${item.name} upload progress`}
              value={item.progress}
            />
          ) : null}
        </li>
      ))}
    </ul>
  )
}
