import { OTPField } from "@base-ui/react/otp-field";
import { Field } from "../field/field";
export interface InputOTPProps {
  label?: string;
  description?: string;
  error?: string;
  length?: number;
  value?: string;
  onValueChange?: (value: string) => void;
  disabled?: boolean;
}
export function InputOTP({
  description,
  disabled,
  error,
  label,
  length = 6,
  onValueChange,
  value,
}: InputOTPProps) {
  const control = (
    <OTPField.Root
      length={length}
      value={value}
      onValueChange={onValueChange}
      disabled={disabled}
      className="flex gap-2"
    >
      <OTPField.Input
        render={<input />}
        maxLength={length}
        className="h-12 w-full rounded-xl border border-border bg-surface-control text-center text-lg tracking-[0.5em] outline-none transition-[background-color,border-color,box-shadow] duration-200 hover:bg-surface-container-highest/40 focus:border-primary focus:bg-surface-card focus:ring-2 focus:ring-ring/20"
      />
    </OTPField.Root>
  );
  return label ? (
    <Field label={label} description={description} error={error}>
      {control}
    </Field>
  ) : (
    control
  );
}
