import type { ReactNode } from 'react'

export type PageWidth = 'readable' | 'standard' | 'wide' | 'full'

const widthClasses: Record<PageWidth, string> = {
  readable: 'max-w-[60rem]',
  standard: 'max-w-[82rem]',
  wide: 'max-w-[94rem]',
  full: 'max-w-none',
}

export function PageContainer({
  children,
  width = 'standard',
}: {
  children: ReactNode
  width?: PageWidth
}) {
  return <div className={`mx-auto w-full ${widthClasses[width]}`}>{children}</div>
}
