import { Slider as BaseSlider } from '@base-ui/react/slider'
import { Field } from '../field/field'
export interface SliderProps {
  label?: string
  description?: string
  error?: string
  value?: number | number[]
  defaultValue?: number | number[]
  min?: number
  max?: number
  step?: number
  marks?: number[]
  onValueChange?: (value: number | number[]) => void
}
export function Slider({
  description,
  error,
  label,
  marks,
  onValueChange,
  ...props
}: SliderProps) {
  const control = (
    <BaseSlider.Root
      onValueChange={onValueChange}
      {...props}
      className="flex h-8 w-full cursor-pointer items-center"
    >
      <BaseSlider.Control className="relative h-2 w-full rounded-full bg-surface-container">
        <BaseSlider.Track className="h-full">
          <BaseSlider.Indicator className="h-full rounded-full bg-primary" />
          <BaseSlider.Thumb className="size-6 cursor-pointer rounded-full border-2 border-primary bg-surface-card shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
        </BaseSlider.Track>
      </BaseSlider.Control>
      {marks?.map((mark) => (
        <span
          key={mark}
          aria-hidden="true"
          style={{ position: "absolute", width: 1, height: 1, overflow: "hidden", clip: "rect(0 0 0 0)", whiteSpace: "nowrap" }}
        >
          Mark {mark}
        </span>
      ))}
    </BaseSlider.Root>
  )
  return label ? (
    <Field label={label} description={description} error={error}>
      {control}
    </Field>
  ) : (
    control
  )
}
