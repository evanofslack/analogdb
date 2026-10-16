"use client";

import { lineColors, monthLabel, pct, Stats, TrendItem } from "@lib/stats";
import { LineChart } from "@mantine/charts";
import { SegmentedControl } from "@mantine/core";
import type { AnalogdbStatsMover } from "analogdb-generated";
import Link from "next/link";
import { useState } from "react";
import Sparkline from "./sparkline";
import shared from "./stats.module.css";
import styles from "./trendSection.module.css";

const chartItems = 8;
const themeMultiples = 6;
const partialSuffix = "__partial";

type Kind = "films" | "cameras" | "filmBrands" | "cameraBrands";

const kinds: { value: Kind; label: string }[] = [
  { value: "films", label: "Films" },
  { value: "cameras", label: "Cameras" },
  { value: "filmBrands", label: "Film brands" },
  { value: "cameraBrands", label: "Camera brands" },
];

function chartData(items: TrendItem[], periods: string[], partial: boolean) {
  const last = periods.length - 1;
  return periods.map((period, i) => {
    const row: Record<string, string | number | null> = { period };
    for (const item of items) {
      const value = item.shares?.[i] ?? 0;
      const name = item.name ?? "";
      row[name] = partial && i === last ? null : value;
      row[name + partialSuffix] = partial && i >= last - 1 ? value : null;
    }
    return row;
  });
}

function MoverList({
  title,
  movers,
  lines,
  kind,
  partial,
}: {
  title: string;
  movers: AnalogdbStatsMover[];
  lines: TrendItem[];
  kind: "films" | "cameras";
  partial: boolean;
}) {
  const bySlug = new Map(lines.map((l) => [l.slug, l]));
  return (
    <div>
      <div className={shared.subheading}>{title}</div>
      <ul className={styles.movers}>
        {movers.map((m) => (
          <li key={m.slug}>
            <Link
              href={`/${kind}/${m.slug}`}
              prefetch={false}
              className={styles.moverName}
            >
              {(m.name ?? "").toUpperCase()}
            </Link>
            <span className={styles.moverChange}>
              {pct(m.from)} → {pct(m.to)}
            </span>
            <Sparkline
              values={bySlug.get(m.slug)?.shares ?? []}
              partial={partial}
            />
          </li>
        ))}
      </ul>
    </div>
  );
}

export default function TrendSection({ stats }: { stats: Stats }) {
  const [kind, setKind] = useState<Kind>("films");
  const [highlight, setHighlight] = useState<string | null>(null);
  const trends = stats.trends ?? {};
  const periods = trends.periods ?? [];
  const partial = (trends.partial ?? []).includes(periods[periods.length - 1]);
  const items = (trends[kind] ?? []).slice(0, chartItems);
  const moverKind =
    kind === "films" || kind === "filmBrands" ? "films" : "cameras";
  const movers = trends.movers?.[moverKind];
  const lines = trends[moverKind] ?? [];
  const base = trends.movers?.base;
  const recent = trends.movers?.recent;
  const top = trends.movers?.films?.up?.[0];

  const series = items.flatMap((item, i) => {
    const color = lineColors[i % lineColors.length];
    const name = item.name ?? "";
    return [
      { name, label: name.toUpperCase(), color },
      {
        name: name + partialSuffix,
        label: name.toUpperCase(),
        color,
        strokeDasharray: "4 3",
      },
    ];
  });

  const complete = partial ? periods.length - 2 : periods.length - 1;
  const themeLines = [...(trends.themes ?? [])]
    .map((line) => {
      const shares = line.shares ?? [];
      return { line, first: shares[0] ?? 0, last: shares[complete] ?? 0 };
    })
    .sort((a, b) => Math.abs(b.last - b.first) - Math.abs(a.last - a.first))
    .slice(0, themeMultiples);

  return (
    <section className={shared.section}>
      <h2 className={shared.heading}>TRENDS</h2>
      <p className={shared.lead}>
        {top
          ? `${(top.name ?? "").toUpperCase()} went from ${pct(
              top.from
            )} to ${pct(
              top.to
            )} of film photos, the biggest rise in the last year.`
          : "What photos in AnalogDB were shot on, half year by half year."}
      </p>
      <SegmentedControl
        size="xs"
        value={kind}
        onChange={(value) => setKind(value as Kind)}
        data={kinds}
        className={styles.toggle}
      />
      <div className={styles.chart}>
        <LineChart
          h={280}
          data={chartData(items, periods, partial)}
          dataKey="period"
          series={series}
          curveType="linear"
          withDots={false}
          withLegend={false}
          gridAxis="y"
          strokeWidth={1.6}
          connectNulls={false}
          unit="%"
          valueFormatter={(v) => v.toFixed(1)}
          yAxisProps={{ width: 40 }}
          lineProps={(s) => ({
            isAnimationActive: false,
            strokeOpacity:
              highlight && s.name.replace(partialSuffix, "") !== highlight
                ? 0.15
                : 1,
          })}
          tooltipProps={{
            content: ({ label, payload }) => {
              const seen = new Set<string>();
              const rows = (payload ?? [])
                .filter((p) => p.value !== null && p.value !== undefined)
                .map((p) => ({
                  name: String(p.name ?? "").replace(partialSuffix, ""),
                  value: Number(p.value),
                  color: String(p.color ?? ""),
                }))
                .filter((r) =>
                  seen.has(r.name) ? false : (seen.add(r.name), true)
                )
                .sort((a, b) => b.value - a.value);
              return (
                <div className={styles.tooltip}>
                  <div className={styles.tooltipLabel}>
                    {label}
                    {partial && label === periods[periods.length - 1]
                      ? " (so far)"
                      : ""}
                  </div>
                  {rows.map((r) => (
                    <div key={r.name} className={styles.tooltipRow}>
                      <span style={{ background: r.color }} />
                      {r.name.toUpperCase()}
                      <b>{r.value.toFixed(1)}%</b>
                    </div>
                  ))}
                </div>
              );
            },
          }}
        />
      </div>
      <ul className={styles.legend}>
        {items.map((item, i) => (
          <li
            key={item.name}
            onMouseEnter={() => setHighlight(item.name ?? null)}
            onMouseLeave={() => setHighlight(null)}
          >
            <span style={{ background: lineColors[i % lineColors.length] }} />
            {(item.name ?? "").toUpperCase()}
          </li>
        ))}
      </ul>
      <p className={shared.note}>
        Share of photos with a known {moverKind === "films" ? "film" : "camera"}
        , each photographer counted at most 5 times per half year.
        {partial && " The dashed part is the half year so far."}
      </p>

      {movers && (
        <>
          <div className={styles.moverGrid}>
            <MoverList
              title="rising"
              movers={movers.up ?? []}
              lines={lines}
              kind={moverKind}
              partial={partial}
            />
            <MoverList
              title="falling"
              movers={movers.down ?? []}
              lines={lines}
              kind={moverKind}
              partial={partial}
            />
          </div>
          {base && recent && (
            <p className={shared.note}>
              {monthLabel(recent.from)} to {monthLabel(recent.to)} compared with{" "}
              {monthLabel(base.from)} to {monthLabel(base.to)}.
            </p>
          )}
        </>
      )}

      {(trends.arrivals ?? []).length > 0 && (
        <>
          <div className={shared.subheading}>new on the scene</div>
          <ul className={styles.arrivals}>
            {(trends.arrivals ?? []).map((a) => (
              <li key={`${a.kind}-${a.slug}`}>
                <Link
                  href={`/${a.kind === "camera" ? "cameras" : "films"}/${
                    a.slug
                  }`}
                  prefetch={false}
                >
                  {(a.name ?? "").toUpperCase()}
                </Link>
                <span>
                  since {monthLabel(a.first)},{" "}
                  {(a.count ?? 0).toLocaleString("en-US")} photos
                </span>
              </li>
            ))}
          </ul>
        </>
      )}

      {themeLines.length > 0 && (
        <>
          <div className={shared.subheading}>themes over time</div>
          <ul className={styles.themes}>
            {themeLines.map(({ line, first, last }) => (
              <li key={line.key}>
                <span className={styles.themeName}>{line.name}</span>
                <Sparkline
                  values={line.shares ?? []}
                  width={120}
                  height={28}
                  partial={partial}
                />
                <span className={styles.themeChange}>
                  {pct(first)} → {pct(last)}
                </span>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
}
