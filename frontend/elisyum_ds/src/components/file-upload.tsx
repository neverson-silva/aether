import type { InputHTMLAttributes } from 'react'
import type { UseFormRegisterReturn } from 'react-hook-form'
import { Attachment, type AttachmentItem } from './attachment'
import { DragAndDrop } from './drag-and-drop'

export interface FileUploadProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  attachments?: AttachmentItem[]
  onFiles?: (files: File[]) => void
  onRemove?: (id: string) => void
  inputProps?: UseFormRegisterReturn
}

export function FileUpload({
  accept,
  attachments = [],
  className = '',
  inputProps,
  multiple = true,
  onFiles,
  onRemove,
  ...inputAttributes
}: FileUploadProps) {
  return (
    <div className={`grid gap-3 ${className}`}>
      <DragAndDrop
        accept={accept}
        multiple={multiple}
        onFiles={onFiles}
      >
        Choose files to upload
      </DragAndDrop>
      <input
        {...inputAttributes}
        {...inputProps}
        accept={accept}
        className="sr-only"
        multiple={multiple}
        onChange={(event) => {
          inputProps?.onChange?.(event)
          onFiles?.(Array.from(event.target.files ?? []))
        }}
        type="file"
      />
      <Attachment
        items={attachments}
        onRemove={onRemove}
      />
    </div>
  )
}
