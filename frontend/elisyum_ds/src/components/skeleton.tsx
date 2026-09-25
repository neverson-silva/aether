import type { HTMLAttributes } from 'react'

export type SkeletonProps = HTMLAttributes<HTMLDivElement>

export function Skeleton({ className = '', ...props }: SkeletonProps) {
  return (
    <div
      {...props}
      aria-hidden="true"
      className={`motion-safe:animate-pulse motion-reduce:animate-none rounded-md bg-surface-3 ${className}`}
    />
  )
}
