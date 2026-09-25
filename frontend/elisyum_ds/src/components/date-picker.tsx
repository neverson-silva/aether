import { Calendar, type CalendarProps } from './calendar'
export type DatePickerProps = CalendarProps
export function DatePicker(props: DatePickerProps) {
  return <Calendar {...props} />
}
