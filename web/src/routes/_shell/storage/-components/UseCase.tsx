export function UseCase({ icon, label, desc }: { icon: string; label: string; desc: string }) {
  return (
    <div className="flex items-start gap-sm p-sm rounded border border-outline-variant/60">
      <span className="material-symbols-outlined text-[20px] text-primary shrink-0">{icon}</span>
      <div>
        <p className="font-body-md text-body-md text-on-surface">{label}</p>
        <p className="font-body-sm text-body-sm text-on-surface-variant">{desc}</p>
      </div>
    </div>
  );
}
