import { type ImgHTMLAttributes, type ReactNode, useState } from 'react'
import { Spinner } from '../spinner/spinner'

export interface AvatarProps
  extends Omit<ImgHTMLAttributes<HTMLImageElement>, 'loading'> {
  fallback?: ReactNode
  status?: 'online' | 'away' | 'offline'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
}
export function Avatar({
  alt = '',
  className = '',
  fallback,
  onError,
  onLoad,
  size = 'md',
  src,
  status,
  loading = false,
  ...props
}: AvatarProps) {
  const [failed, setFailed] = useState(false)
  const sizes = {
    sm: 'size-8 text-xs',
    md: 'size-10 text-sm',
    lg: 'size-14 text-base',
  }
  return (
    <span
      className={`relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-surface-container font-semibold text-muted-foreground ${sizes[size]} ${className}`}
    >
      {loading ? (
        <Spinner size="sm" />
      ) : src && !failed ? (
        <img
          src={src}
          alt={alt}
          onError={(event) => {
            setFailed(true)
            onError?.(event)
          }}
          onLoad={onLoad}
          {...props}
        />
      ) : (
        <span aria-hidden={alt ? undefined : true}>{fallback}</span>
      )}
      {status ? (
        <span
          className={`absolute bottom-0 right-0 size-2.5 rounded-full border-2 border-surface-card ${status === 'online' ? 'bg-status-success' : status === 'away' ? 'bg-status-warning' : 'bg-muted-foreground'}`}
          aria-hidden="true"
        />
      ) : null}
      {status ? <span className="sr-only">{status}</span> : null}
    </span>
  )
}
