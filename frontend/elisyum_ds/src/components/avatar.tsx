import { useEffect, useState, type ImgHTMLAttributes } from 'react'

export interface AvatarProps extends Omit<ImgHTMLAttributes<HTMLImageElement>, 'alt'> {
  name: string
  alt?: string
  size?: 'sm' | 'md' | 'lg'
}

export function Avatar({
  alt,
  className = '',
  name,
  size = 'md',
  src,
  ...props
}: AvatarProps) {
  const [imageFailed, setImageFailed] = useState(false)
  useEffect(() => setImageFailed(false), [src])
  const sizeClass =
    size === 'sm'
      ? 'size-7 text-label'
      : size === 'lg'
        ? 'size-12 text-body'
        : 'size-9 text-supporting'
  const initials = name
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0])
    .join('')
    .toUpperCase()
  return src && !imageFailed ? (
    <img
      {...props}
      alt={alt ?? name}
      className={`rounded-full object-cover ${sizeClass} ${className}`}
      onError={(event) => {
        setImageFailed(true)
        props.onError?.(event)
      }}
      src={src}
    />
  ) : (
    <span
      aria-label={alt ?? name}
      role="img"
      className={`inline-flex items-center justify-center rounded-full bg-action-soft font-medium text-action-strong ${sizeClass} ${className}`}
    >
      {initials}
    </span>
  )
}
