"use client";

import { popularity, Stats } from "@lib/stats";
import { SegmentedControl } from "@mantine/core";
import { useEffect, useState } from "react";
import HeatGrid, { HeatCell } from "./heatGrid";
import styles from "./postingSection.module.css";
import shared from "./stats.module.css";

const days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
const hours = Array.from({ length: 24 }, (_, h) => String(h));

type Mode = "popularity" | "count";

function zoneLabel(offset: number): string {
  if (offset === 0) return "UTC";
  return `UTC${offset > 0 ? "+" : "−"}${Math.abs(offset)}`;
}

export default function PostingSection({ stats }: { stats: Stats }) {
  const [mode, setMode] = useState<Mode>("popularity");
  // hours east of UTC, 0 until the browser tells us
  const [offset, setOffset] = useState(0);
  useEffect(() => {
    setOffset(Math.round(-new Date().getTimezoneOffset() / 60));
  }, []);

  const local = Array.from({ length: 7 }, () =>
    Array.from({ length: 24 }, () => ({ count: 0, popularity: 0 }))
  );
  for (const cell of stats.posting?.cells ?? []) {
    const utc = ((cell.weekday ?? 1) - 1) * 24 + (cell.hour ?? 0);
    const shifted = (((utc + offset) % 168) + 168) % 168;
    local[Math.floor(shifted / 24)][shifted % 24] = {
      count: cell.count ?? 0,
      popularity: cell.popularity ?? 0,
    };
  }

  const counts = local.flat().map((c) => c.count);
  const maxCount = Math.max(...counts, 1);
  const pops = local
    .flat()
    .filter((c) => c.count > 0)
    .map((c) => c.popularity);
  const minPop = Math.min(...pops, 50);
  const maxPop = Math.max(...pops, 50);

  const rows = local.map((cells, d) => ({
    label: days[d],
    cells: cells.map((c, h): HeatCell => {
      const level =
        c.count === 0
          ? null
          : mode === "count"
          ? c.count / maxCount
          : (c.popularity - minPop) / (maxPop - minPop || 1);
      return {
        level,
        title: `${days[d]} ${h}:00, ${c.count} photos, popularity ${popularity(
          c.popularity
        )}`,
      };
    }),
  }));

  return (
    <section className={shared.section}>
      <h2 className={shared.heading}>WHEN TO POST</h2>
      <p className={shared.lead}>
        When photos were posted, in your time zone: average popularity by hour,
        or how many photos. Darker is higher.
      </p>
      <SegmentedControl
        size="xs"
        value={mode}
        onChange={(value) => setMode(value as Mode)}
        data={[
          { value: "popularity", label: "Popularity" },
          { value: "count", label: "Photos" },
        ]}
        className={styles.toggle}
      />
      <HeatGrid
        columns={hours}
        rows={rows}
        compact
        columnLabel={(column, i) => (i % 3 === 0 ? column : "")}
        ariaLabel="popularity by weekday and hour"
      />
      <p className={shared.note}>Times in {zoneLabel(offset)}.</p>
    </section>
  );
}
