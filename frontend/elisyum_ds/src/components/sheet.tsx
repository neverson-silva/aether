import { Dialog, type DialogProps } from './dialog'

export interface SheetProps extends DialogProps {
  side?: 'top' | 'right' | 'bottom' | 'left'
}

export function Sheet({ side = 'bottom', popupClassName = '', ...props }: SheetProps) {
  const placement =
    side === 'top'
      ? 'inset-x-0 top-0'
      : side === 'right'
        ? 'inset-y-0 right-0 top-0'
        : side === 'left'
          ? 'inset-y-0 left-0 top-0'
          : 'inset-x-0 bottom-0'
  const motion =
    side === 'top'
      ? 'data-[starting-style]:!translate-y-[-100%] data-[ending-style]:!translate-y-[-100%]'
      : side === 'right'
        ? 'data-[starting-style]:!translate-x-full data-[ending-style]:!translate-x-full'
        : side === 'left'
          ? 'data-[starting-style]:!-translate-x-full data-[ending-style]:!-translate-x-full'
          : 'data-[starting-style]:!translate-y-full data-[ending-style]:!translate-y-full'
  const radius =
    side === 'top'
      ? 'rounded-b-xl'
      : side === 'bottom'
        ? 'rounded-t-xl'
        : side === 'left'
          ? 'rounded-r-xl'
          : 'rounded-l-xl'
  return (
    <Dialog
      {...props}
      popupClassName={`fixed ${placement} max-h-[90vh] max-w-none !translate-x-0 !translate-y-0 !transition-transform motion-reduce:!transition-none ${radius} ${motion} !ease-ely-drawer will-change-transform data-[starting-style]:!scale-100 data-[ending-style]:!scale-100 ${popupClassName}`}
    />
  )
}
