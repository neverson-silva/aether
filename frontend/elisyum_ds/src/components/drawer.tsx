import { Dialog, type DialogProps } from './dialog'

export interface DrawerProps extends DialogProps {
  side?: 'left' | 'right'
}

export function Drawer({ side = 'right', popupClassName = '', ...props }: DrawerProps) {
  const motion =
    side === 'left'
      ? 'data-[starting-style]:!-translate-x-full data-[ending-style]:!-translate-x-full'
      : 'data-[starting-style]:!translate-x-full data-[ending-style]:!translate-x-full'
  return (
    <Dialog
      {...props}
      popupClassName={`fixed inset-y-0 ${side === 'left' ? 'left-0 right-auto rounded-r-xl' : 'left-auto right-0 rounded-l-xl'} top-0 max-h-none w-[min(28rem,calc(100vw-1rem))] !translate-y-0 !transition-transform motion-reduce:!transition-none ${motion} !ease-ely-drawer will-change-transform data-[starting-style]:!scale-100 data-[ending-style]:!scale-100 ${popupClassName}`}
    />
  )
}
