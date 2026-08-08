import { twMerge } from "tailwind-merge";

export function cn(...parts: (string | false | null | undefined)[]) {
  return twMerge(parts.filter(Boolean).join(" "));
}

export function SpinnerMini({ className }: { className?: string }) {
  return <span className={cn("inline-block border-2 border-outline-variant border-t-primary rounded-full animate-spin", className ?? "w-4 h-4")} />;
}
