import { Toolbar } from '@base-ui/react/toolbar'
import type { HTMLAttributes } from 'react'

export interface ButtonGroupProps extends HTMLAttributes<HTMLDivElement> {
  orientation?: 'horizontal' | 'vertical'
  attached?: boolean
}
export function ButtonGroup({
  className = '',
  orientation = 'horizontal',
  attached = false,
  ...props
}: ButtonGroupProps) {
  return (
    <Toolbar.Root
      orientation={orientation}
      className={`flex ${orientation === 'vertical' ? 'flex-col' : 'flex-row'} ${attached ? 'gap-0 [&>button:not(:first-child)]:rounded-l-none [&>button:not(:last-child)]:rounded-r-none [&>button:not(:first-child)]:border-l-0' : 'gap-2'} ${className}`}
      {...props}
    />
  )
}
