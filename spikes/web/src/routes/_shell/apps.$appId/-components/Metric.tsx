export function Metric({ label, value, icon }: { label: string; value: string; icon: string }) {
  return (
    <div className="flex flex-col gap-xs">
      <span className="material-symbols-outlined text-[18px] text-primary">{icon}</span>
      <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">{label}</span>
      <span className="font-code-md text-code-md text-on-surface">{value}</span>
    </div>
  );
}
