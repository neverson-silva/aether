import type { ReactNode } from 'react'
import type { FieldValues, UseFormRegister } from 'react-hook-form'
import { Checkbox } from './checkbox'
import { Field } from './field'
import { Input } from './input'
import { Select } from './select'
import { Textarea } from './textarea'

export interface FormFieldDefinition {
  name: string
  label: ReactNode
  type?: 'text' | 'number' | 'textarea' | 'select' | 'checkbox'
  options?: { value: string; label: ReactNode }[]
  required?: boolean
}
export interface FormBuilderProps<T extends FieldValues> {
  fields: FormFieldDefinition[]
  register: UseFormRegister<T>
  errors?: Partial<Record<string, { message?: string }>>
  className?: string
}

export function FormBuilder<T extends FieldValues>({
  className = '',
  errors,
  fields,
  register,
}: FormBuilderProps<T>) {
  return (
    <div className={`grid gap-4 ${className}`}>
      {fields.map((field) => (
        <Field
          id={field.name}
          key={field.name}
          label={field.label}
          required={field.required}
          error={errors?.[field.name]?.message}
        >
          {field.type === 'textarea' ? (
            <Textarea
              id={field.name}
              {...register(field.name as never)}
            />
          ) : field.type === 'select' ? (
            <Select
              id={field.name}
              {...register(field.name as never)}
            >
              <option value="">Choose an option</option>
              {field.options?.map((option) => (
                <option
                  key={option.value}
                  value={option.value}
                >
                  {option.label}
                </option>
              ))}
            </Select>
          ) : field.type === 'checkbox' ? (
            <Checkbox
              id={field.name}
              {...register(field.name as never)}
            />
          ) : (
            <Input
              id={field.name}
              type={field.type === 'number' ? 'number' : 'text'}
              {...register(field.name as never)}
            />
          )}
        </Field>
      ))}
    </div>
  )
}
