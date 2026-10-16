"use client";

import {
  familyColors,
  filmName,
  lookSectionId,
  StatsFilm,
  statsParsers,
} from "@lib/stats";
import type { AnalogdbStatsFamily } from "analogdb-generated";
import { useQueryStates } from "nuqs";
import { useState } from "react";
import styles from "./familySection.module.css";
import shared from "./stats.module.css";

const labelled = 12;
const pad = 9;
const maxRadius = 4.2;
// approximate width of one label character in chart units
const labelWidth = 1.45;

interface FamilySectionProps {
  films: StatsFilm[];
  families: AnalogdbStatsFamily[];
}

export default function FamilySection({ films, families }: FamilySectionProps) {
  const [, setParams] = useQueryStates(statsParsers);
  const [hovered, setHovered] = useState<string | null>(null);

  const placed = films.filter((f) => f.position);
  const largest = Math.max(...placed.map((f) => f.postCount ?? 0), 1);
  const x = (v: number) => pad + ((v + 1) / 2) * (100 - 2 * pad);
  const y = (v: number) => 100 - (pad + ((v + 1) / 2) * (100 - 2 * pad));
  const radius = (film: StatsFilm) =>
    Math.max(0.9, maxRadius * Math.sqrt((film.postCount ?? 0) / largest));

  // label the largest films whose label doesn't overlap one already placed
  const named = new Set<string | undefined>();
  const boxes: { x0: number; x1: number; y0: number; y1: number }[] = [];
  for (const film of [...placed].sort(
    (a, b) => (b.postCount ?? 0) - (a.postCount ?? 0)
  )) {
    if (named.size === labelled) break;
    const cx = x(film.position?.x ?? 0);
    const cy = y(film.position?.y ?? 0);
    const x0 = cx + radius(film) + 0.8;
    const box = {
      x0,
      x1: x0 + filmName(film).length * labelWidth,
      y0: cy - 1.2,
      y1: cy + 1.4,
    };
    const hit = boxes.some(
      (b) => box.x0 < b.x1 && box.x1 > b.x0 && box.y0 < b.y1 && box.y1 > b.y0
    );
    if (!hit) {
      boxes.push(box);
      named.add(film.slug);
    }
  }
  const ektar = films.find((f) => f.slug === "kodak-ektar-100");

  const open = (slug: string | undefined) => {
    if (!slug) return;
    setParams({ film: slug });
    document
      .getElementById(lookSectionId)
      ?.scrollIntoView({ behavior: "smooth" });
  };

  // smaller dots on top, so a large film never hides a small one
  const ordered = [...placed].sort(
    (a, b) => (b.postCount ?? 0) - (a.postCount ?? 0)
  );

  return (
    <section className={shared.section}>
      <h2 className={shared.heading}>FILM FAMILIES</h2>
      <p className={shared.lead}>
        Films placed by how alike their photos look, closer means more alike.
        {ektar?.family === "landscape" && " Ektar sits with the slide films."}
      </p>
      <div className={styles.chart}>
        <svg
          viewBox="0 0 100 100"
          role="img"
          aria-label="map of films by how alike their photos look"
        >
          {ordered.map((film) => {
            const pos = film.position!;
            const r = radius(film);
            const cx = x(pos.x ?? 0);
            const cy = y(pos.y ?? 0);
            const active = hovered === film.slug;
            const showLabel = named.has(film.slug) || active;
            return (
              <g
                key={film.slug}
                className={styles.dot}
                onMouseEnter={() => setHovered(film.slug ?? null)}
                onMouseLeave={() => setHovered(null)}
                onClick={() => open(film.slug)}
              >
                <title>{filmName(film)}</title>
                <circle
                  cx={cx}
                  cy={cy}
                  r={r}
                  fill={familyColors[film.family ?? ""] ?? familyColors.bw}
                  fillOpacity={active ? 1 : 0.75}
                  stroke={active ? "var(--adb-heading)" : "var(--adb-surface)"}
                  strokeWidth={0.3}
                />
                {showLabel && (
                  <text
                    x={cx + r + 0.8}
                    y={cy + 1}
                    className={`${styles.label} ${
                      active ? styles.labelActive : ""
                    }`}
                  >
                    {filmName(film)}
                  </text>
                )}
              </g>
            );
          })}
        </svg>
      </div>
      <ul className={styles.legend}>
        {families.map((family) => (
          <li key={family.key}>
            <span
              style={{
                background: familyColors[family.key ?? ""] ?? familyColors.bw,
              }}
            />
            {family.name}
          </li>
        ))}
      </ul>
    </section>
  );
}
