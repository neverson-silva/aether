import { forwardRef, type InputHTMLAttributes } from 'react'
import type { ComboboxOption } from './combobox'
import { Combobox } from './combobox'

export interface SelectSearchProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'list'> {
  options: ComboboxOption[]
}

export const SelectSearch = forwardRef<HTMLInputElement, SelectSearchProps>(
  function SelectSearchView(props, ref) {
    return (
      <Combobox
        {...props}
        ref={ref}
      />
    )
  },
)
