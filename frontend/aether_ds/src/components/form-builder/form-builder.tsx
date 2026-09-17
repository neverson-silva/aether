import { DotsSixVertical, Trash } from "@phosphor-icons/react";
import { type ReactNode, useState } from "react";
export interface FormFieldDefinition {
  id: string;
  label: string;
  type: "text" | "number" | "select" | "secret";
  required?: boolean;
  options?: string[];
  description?: string;
}
export interface FormBuilderProps {
  fields: FormFieldDefinition[];
  onChange?: (fields: FormFieldDefinition[]) => void;
  preview?: boolean;
  children?: ReactNode;
}
export function FormBuilder({
  children,
  fields: initialFields,
  onChange,
  preview,
}: FormBuilderProps) {
  const [fields, setFields] = useState(initialFields);
  const update = (next: FormFieldDefinition[]) => {
    setFields(next);
    onChange?.(next);
  };
  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <div className="space-y-2">
        {fields.map((field, index) => (
          <div
            key={field.id}
            className="flex items-start gap-2 rounded-2xl border border-border bg-surface-card p-3 shadow-sm transition-[border-color,box-shadow] duration-200 hover:border-primary/40 hover:shadow-md"
          >
            <DotsSixVertical
              size={18}
              className="mt-1 shrink-0 cursor-grab text-muted-foreground"
            />
            <div className="min-w-0 flex-1">
              <div className="text-body-sm font-semibold text-foreground">
                {field.label}
              </div>
              <div className="text-body-sm text-muted-foreground">
                {field.type}
                {field.required ? " · required" : ""}
              </div>
            </div>
            <button
              type="button"
              aria-label={`Remove ${field.label}`}
              onClick={() =>
                update(fields.filter((_, itemIndex) => itemIndex !== index))
              }
              className="rounded-lg p-1 text-muted-foreground outline-none transition-[background-color,color,transform] duration-150 hover:bg-status-danger/10 hover:text-status-danger focus-visible:ring-2 focus-visible:ring-ring active:scale-95"
            >
              <Trash size={16} />
            </button>
          </div>
        ))}
      </div>
      {preview ? (
        <div className="rounded-2xl border border-border bg-surface-card p-5 shadow-sm">
          <div className="mb-4 text-body-sm font-semibold">Preview</div>
          {fields.map((field) => (
            <label
              key={field.id}
              className="mb-4 block text-body-sm font-semibold"
            >
              {field.label}
              {field.required ? " *" : ""}
              <input
                type={field.type === "secret" ? "password" : field.type}
                className="mt-1 h-10 w-full rounded-xl border border-border bg-surface-control px-3 font-normal outline-none transition-[background-color,border-color,box-shadow] duration-200 hover:bg-surface-container-highest/40 focus:border-primary focus:bg-surface-card focus:ring-2 focus:ring-primary/20"
              />
              {field.description ? (
                <span className="mt-1 block text-body-sm font-normal text-muted-foreground">
                  {field.description}
                </span>
              ) : null}
            </label>
          ))}
        </div>
      ) : (
        children
      )}
    </div>
  );
}
