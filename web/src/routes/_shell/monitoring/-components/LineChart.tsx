import { useMemo } from "react";
import { cn } from "../../../../components/ui";

export interface ChartSeries {
  name: string;
  color: string;
  points: number[];
}

// Lightweight multi-series SVG line chart. No chart library is used; this
// follows the existing inline-SVG sparkline convention and scales with the
// design system. Every chart answers one question (trend), not decoration.
export function LineChart({
  series,
  max,
  height = 120,
  className,
  unit,
}: {
  series: ChartSeries[];
  max?: number;
  height?: number;
  className?: string;
  unit?: (v: number) => string;
}) {
  const W = 100;
  const H = 48;
  const len = Math.max(0, ...series.map((s) => s.points.length));

  const { scaleMax, path } = useMemo(() => {
    const all = series.flatMap((s) => s.points);
    let m = max ?? 100;
    if (all.length > 0) {
      const dataMax = Math.max(...all);
      if (dataMax > m) m = dataMax * 1.1;
    }
    if (m <= 0) m = 1;
    const step = len > 1 ? W / (len - 1) : 0;
    const p = (pts: number[]) =>
      "M" + pts.map((v, i) => `${(i * step).toFixed(2)},${(H - (Math.min(Math.max(v, 0), m) / m) * H).toFixed(2)}`).join(" L");
    return { scaleMax: m, path: p };
  }, [series, max, len]);

  if (len < 2) {
    return (
      <div className={cn("flex items-center justify-center text-on-surface-variant/40 text-[11px]", className)} style={{ height }}>
        collecting…
      </div>
    );
  }

  return (
    <div className={className}>
      <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" style={{ height, width: "100%" }}>
        {[0.25, 0.5, 0.75].map((f) => (
          <line key={f} x1={0} x2={W} y1={H * f} y2={H * f} stroke="rgba(255,255,255,0.06)" strokeWidth={0.2} vectorEffect="non-scaling-stroke" />
        ))}
        {series.map((s) => (
          <path key={s.name} d={path(s.points)} fill="none" stroke={s.color} strokeWidth={0.7} vectorEffect="non-scaling-stroke" strokeLinejoin="round" />
        ))}
      </svg>
      {unit && series.some((s) => s.points.length > 0) && (
        <div className="flex items-center justify-between font-code-md text-code-md text-on-surface-variant/50 mt-xs">
          <span>{unit(0)}</span>
          <span>{unit(scaleMax)}</span>
        </div>
      )}
    </div>
  );
}
