import {
  type ChangeEvent,
  type DragEvent,
  type LabelHTMLAttributes,
  useState,
} from "react";
export interface FileUploadProps extends Omit<
  LabelHTMLAttributes<HTMLLabelElement>,
  "onError"
> {
  accept?: string;
  multiple?: boolean;
  maxSize?: number;
  onFilesChange?: (files: File[]) => void;
  onError?: (message: string) => void;
  disabled?: boolean;
}
export function FileUpload({
  accept,
  className = "",
  disabled,
  maxSize,
  multiple,
  onError,
  onFilesChange,
  ...props
}: FileUploadProps) {
  const [dragging, setDragging] = useState(false);
  const handleFiles = (files: FileList | null) => {
    if (!files) return;
    const next = Array.from(files);
    const invalid = maxSize
      ? next.find((file) => file.size > maxSize)
      : undefined;
    if (invalid) {
      onError?.(`${invalid.name} exceeds the maximum file size`);
      return;
    }
    onFilesChange?.(next);
  };
  const onDrop = (event: DragEvent<HTMLLabelElement>) => {
    event.preventDefault();
    setDragging(false);
    handleFiles(event.dataTransfer.files);
  };
  return (
    <label
      className={`rounded-2xl border-2 border-dashed bg-surface-card p-6 text-center shadow-sm outline-none transition-[border-color,background-color,box-shadow,transform] duration-200 focus-within:ring-2 focus-within:ring-ring ${dragging ? "border-primary bg-primary/10 shadow-md" : "border-border hover:border-primary/60 hover:bg-surface-container-low"} ${disabled ? "pointer-events-none opacity-50" : ""} ${className}`}
      htmlFor="file-upload"
      onDragOver={(event) => {
        event.preventDefault();
        setDragging(true);
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={onDrop}
      {...props}
    >
      <input
        id="file-upload"
        type="file"
        accept={accept}
        multiple={multiple}
        disabled={disabled}
        className="sr-only"
        onChange={(event: ChangeEvent<HTMLInputElement>) =>
          handleFiles(event.target.files)
        }
      />
      <span className="cursor-pointer text-body-sm text-muted-foreground">
        Drop files here or{" "}
        <span className="font-semibold text-primary">browse</span>
      </span>
    </label>
  );
}
