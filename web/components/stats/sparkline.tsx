interface SparklineProps {
  values: number[];
  width?: number;
  height?: number;
  // the last point is a period still in progress, drawn dashed
  partial?: boolean;
}

export default function Sparkline({
  values,
  width = 60,
  height = 18,
  partial,
}: SparklineProps) {
  if (values.length < 2)
    return <svg width={width} height={height} aria-hidden />;
  const max = Math.max(...values);
  const min = Math.min(...values);
  const span = max - min || 1;
  const points = values.map((v, i) => [
    (i / (values.length - 1)) * (width - 2) + 1,
    height - 1 - ((v - min) / span) * (height - 2),
  ]);
  const path = (pts: number[][]) =>
    pts
      .map(
        ([x, y], i) => `${i === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`
      )
      .join(" ");
  const solid = partial ? points.slice(0, -1) : points;
  return (
    <svg width={width} height={height} aria-hidden>
      <path
        d={path(solid)}
        fill="none"
        stroke="var(--adb-muted)"
        strokeWidth={1.2}
      />
      {partial && (
        <path
          d={path(points.slice(-2))}
          fill="none"
          stroke="var(--adb-muted)"
          strokeWidth={1.2}
          strokeDasharray="2 2"
        />
      )}
    </svg>
  );
}
