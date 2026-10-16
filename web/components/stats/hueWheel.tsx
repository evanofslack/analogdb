import { hueColor } from "@lib/stats";

interface HueWheelProps {
  hue: number[];
  // another film's bins, drawn as an outline on top
  compare?: number[];
  size: number;
  label?: string;
}

const innerRadius = 0.12;

function wedgePath(
  bins: number,
  index: number,
  radius: number,
  max: number
): string {
  const step = (2 * Math.PI) / bins;
  // hue 0 points right, hues run counter clockwise like the color wheel
  const a0 = -index * step;
  const a1 = -(index + 1) * step;
  const r0 = innerRadius;
  const r1 = innerRadius + (1 - innerRadius) * radius;
  if (r1 <= r0) return "";
  const point = (r: number, a: number) =>
    `${(r * Math.cos(a) * max).toFixed(2)} ${(r * Math.sin(a) * max).toFixed(
      2
    )}`;
  return [
    `M ${point(r0, a0)}`,
    `L ${point(r1, a0)}`,
    `A ${(r1 * max).toFixed(2)} ${(r1 * max).toFixed(2)} 0 0 0 ${point(
      r1,
      a1
    )}`,
    `L ${point(r0, a1)}`,
    `A ${(r0 * max).toFixed(2)} ${(r0 * max).toFixed(2)} 0 0 1 ${point(
      r0,
      a0
    )}`,
    "Z",
  ].join(" ");
}

// radii scale with the square root of each bin's share, so area tracks share
function radii(bins: number[], peak: number): number[] {
  return bins.map((value) => (peak > 0 ? Math.sqrt(value / peak) : 0));
}

export default function HueWheel({ hue, compare, size, label }: HueWheelProps) {
  const bins = hue.length;
  const half = size / 2;
  const peak = Math.max(...hue, ...(compare ?? []), 0);
  const main = radii(hue, peak);
  const outline = compare ? radii(compare, peak) : null;
  const step = 360 / Math.max(bins, 1);
  const ring = innerRadius + (1 - innerRadius) * Math.sqrt(0.25);

  return (
    <svg
      width={size}
      height={size}
      viewBox={`${-half} ${-half} ${size} ${size}`}
      role="img"
      aria-label={label}
    >
      <circle
        r={ring * (half - 1)}
        fill="none"
        stroke="var(--adb-border-soft)"
        strokeWidth={1}
      />
      {main.map((radius, i) => (
        <path
          key={i}
          d={wedgePath(bins, i, radius, half - 1)}
          fill={hueColor((i + 0.5) * step)}
        />
      ))}
      {outline?.map((radius, i) => (
        <path
          key={`c${i}`}
          d={wedgePath(bins, i, radius, half - 1)}
          fill="none"
          stroke="var(--adb-heading)"
          strokeWidth={1}
        />
      ))}
    </svg>
  );
}
