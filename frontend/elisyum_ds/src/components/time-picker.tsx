import { forwardRef, type InputHTMLAttributes } from 'react'
import { inputControlClasses } from './input-styles'
export type TimePickerProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>
export const TimePicker = forwardRef<HTMLInputElement, TimePickerProps>(
  function TimePickerView({ className = '', ...props }, ref) {
    return (
      <input
        {...props}
        ref={ref}
        type="time"
        className={`${inputControlClasses} ${className}`}
      />
    )
  },
)
