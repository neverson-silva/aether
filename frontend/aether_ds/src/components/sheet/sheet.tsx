import type { ComponentProps } from 'react'
import { Drawer } from '../drawer/drawer'
export interface SheetProps extends ComponentProps<typeof Drawer> {}
export function Sheet(props: SheetProps) {
  return <Drawer {...props} />
}
