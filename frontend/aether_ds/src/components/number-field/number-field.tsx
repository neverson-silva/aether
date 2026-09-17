import { NumberField as BaseNumberField } from "@base-ui/react/number-field";
import { Field } from "../field/field";
export interface NumberFieldProps {
  label?: string;
  description?: string;
  error?: string;
  value?: number;
  defaultValue?: number;
  min?: number;
  max?: number;
  step?: number;
  onValueChange?: (value: number | null) => void;
}
export function NumberField({
  description,
  error,
  label,
  ...props
}: NumberFieldProps) {
  const control = (
    <BaseNumberField.Root {...props}>
      <BaseNumberField.Group className="flex h-10 overflow-hidden rounded-xl border border-border bg-surface-control shadow-sm transition-[border-color,box-shadow] duration-200 focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20">
        <BaseNumberField.Decrement className="w-10 text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring active:scale-[0.94]">
          −
        </BaseNumberField.Decrement>
        <BaseNumberField.Input className="min-w-0 flex-1 bg-transparent text-center outline-none" />
        <BaseNumberField.Increment className="w-10 text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-surface-container hover:text-foreground focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring active:scale-[0.94]">
          +
        </BaseNumberField.Increment>
      </BaseNumberField.Group>
    </BaseNumberField.Root>
  );
  return label ? (
    <Field label={label} description={description} error={error}>
      {control}
    </Field>
  ) : (
    control
  );
}
