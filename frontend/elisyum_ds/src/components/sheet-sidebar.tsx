import type { DialogProps } from './dialog'
import { Sheet } from './sheet'

export type SheetSidebarProps = DialogProps

export function SheetSidebar(props: SheetSidebarProps) {
  return (
    <Sheet
      {...props}
      side="left"
      popupClassName="w-[min(22rem,calc(100vw-1rem))]"
    />
  )
}
