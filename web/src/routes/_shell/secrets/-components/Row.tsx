export function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between p-sm rounded border border-outline-variant/60">
      <span className="font-body-sm text-body-sm text-on-surface">{label}</span>
      <span className="font-code-md text-code-md text-on-surface-variant">{value}</span>
    </div>
  );
}
