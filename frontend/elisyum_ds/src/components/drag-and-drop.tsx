import { UploadSimple } from '@phosphor-icons/react'
import { useRef, useState, type DragEvent, type ReactNode } from 'react'

export interface DragAndDropProps {
  children?: ReactNode
  onFiles?: (files: File[]) => void
  accept?: string
  multiple?: boolean
  className?: string
}

export function DragAndDrop({
  accept,
  children,
  className = '',
  multiple = true,
  onFiles,
}: DragAndDropProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [active, setActive] = useState(false)
  const handleFiles = (files: FileList | null) => {
    if (files) onFiles?.(Array.from(files))
  }
  const handleDrop = (event: DragEvent<HTMLElement>) => {
    event.preventDefault()
    setActive(false)
    handleFiles(event.dataTransfer.files)
  }
  return (
    <>
      <button
        aria-describedby="elysium-upload-help"
        className={`group grid w-full cursor-pointer justify-items-center gap-2 rounded-lg border border-dashed p-8 text-center font-inherit transition-[background-color,border-color,box-shadow,transform] duration-[var(--ely-duration-fast)] motion-reduce:transition-none focus-visible:border-focus focus-visible:ring-2 focus-visible:ring-focus/25 ${active ? 'border-focus bg-action-soft shadow-elevation-1' : 'border-border-default bg-surface-1 hover:border-focus hover:bg-surface-interactive'} ${className}`}
        onClick={() => inputRef.current?.click()}
        onDragEnter={(event) => {
          event.preventDefault()
          setActive(true)
        }}
        onDragLeave={() => setActive(false)}
        onDragOver={(event) => event.preventDefault()}
        onDrop={handleDrop}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault()
            inputRef.current?.click()
          }
        }}
        type="button"
      >
        <span
          className={`grid size-10 place-items-center rounded-md border border-border-default bg-surface-2 text-action transition-[background-color,border-color,transform] duration-[var(--ely-duration-fast)] group-hover:border-action group-hover:bg-action-soft motion-safe:group-active:scale-[var(--ely-motion-press-scale)] ${active ? 'border-action bg-action text-on-action motion-safe:scale-110' : ''}`}
        >
          <UploadSimple
            aria-hidden="true"
            size={22}
          />
        </span>
        <span className="text-body text-text-primary">
          {children ?? 'Drop files here or browse'}
        </span>
        <span
          className="text-supporting text-text-tertiary"
          id="elysium-upload-help"
        >
          Files are validated before upload.
        </span>
      </button>
      <input
        accept={accept}
        className="sr-only"
        multiple={multiple}
        onChange={(event) => handleFiles(event.target.files)}
        ref={inputRef}
        type="file"
      />
    </>
  )
}
