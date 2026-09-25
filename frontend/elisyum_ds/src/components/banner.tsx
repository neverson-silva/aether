import type { HTMLAttributes, ReactNode } from 'react'
import { Alert, type AlertProps } from './alert'

export interface BannerProps extends Omit<AlertProps, 'title'> {
  title: ReactNode
}

export function Banner({ className = '', ...props }: BannerProps) {
  return (
    <Alert
      {...props}
      className={`rounded-none border-x-0 border-t-0 ${className}`}
    />
  )
}
