import type { HTMLAttributes, ReactNode } from 'react'

export interface BoxProps extends HTMLAttributes<HTMLDivElement> {
  children?: ReactNode
}
export function Box({ children, className = '', ...props }: BoxProps) {
  return (
    <div
      {...props}
      className={className}
    >
      {children}
    </div>
  )
}
export interface StackProps extends BoxProps {
  gap?: '1' | '2' | '3' | '4' | '6' | '8'
}
const gapClasses = {
  '1': 'gap-1',
  '2': 'gap-2',
  '3': 'gap-3',
  '4': 'gap-4',
  '6': 'gap-6',
  '8': 'gap-8',
} as const
const columnClasses = {
  1: 'md:grid-cols-1',
  2: 'md:grid-cols-2',
  3: 'md:grid-cols-3',
  4: 'md:grid-cols-4',
  6: 'md:grid-cols-6',
} as const
export function Stack({ children, className = '', gap = '4', ...props }: StackProps) {
  return (
    <div
      {...props}
      className={`grid ${gapClasses[gap]} ${className}`}
    >
      {children}
    </div>
  )
}
export interface InlineProps extends BoxProps {
  gap?: '1' | '2' | '3' | '4' | '6' | '8'
  wrap?: boolean
}
export function Inline({
  children,
  className = '',
  gap = '2',
  wrap = true,
  ...props
}: InlineProps) {
  return (
    <div
      {...props}
      className={`flex items-center ${gapClasses[gap]} ${wrap ? 'flex-wrap' : ''} ${className}`}
    >
      {children}
    </div>
  )
}
export interface GridProps extends BoxProps {
  columns?: 1 | 2 | 3 | 4 | 6
}
export function Grid({ children, className = '', columns = 2, ...props }: GridProps) {
  return (
    <div
      {...props}
      className={`grid grid-cols-1 gap-4 ${columnClasses[columns]} ${className}`}
    >
      {children}
    </div>
  )
}
export function Container({ children, className = '', ...props }: BoxProps) {
  return (
    <div
      {...props}
      className={`mx-auto w-full max-w-7xl px-4 sm:px-6 ${className}`}
    >
      {children}
    </div>
  )
}
export function Bleed({ children, className = '', ...props }: BoxProps) {
  return (
    <div
      {...props}
      className={`-mx-4 sm:-mx-6 ${className}`}
    >
      {children}
    </div>
  )
}
export function Divider({ className = '', ...props }: Omit<BoxProps, 'children'>) {
  return (
    <div
      {...props}
      aria-hidden="true"
      className={`h-px w-full bg-border-subtle ${className}`}
    />
  )
}
export function VisuallyHidden({ children, ...props }: BoxProps) {
  return (
    <span
      {...props}
      className="sr-only"
    >
      {children}
    </span>
  )
}
