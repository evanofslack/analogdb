import { filmName, galleryHref, Stats } from "@lib/stats";
import Link from "next/link";
import HeatGrid, { HeatCell } from "./heatGrid";
import shared from "./stats.module.css";

// lift maps to color on a log scale, 1× is neutral and 8× is full
function liftLevel(lift: number): number {
  return Math.max(0, Math.min(3, Math.log2(lift))) / 3;
}

export default function SubjectSection({ stats }: { stats: Stats }) {
  const words = stats.subjects?.words ?? [];
  const bySlug = new Map((stats.films ?? []).map((f) => [f.slug, f]));
  const rows = (stats.subjects?.rows ?? []).flatMap((row) => {
    const film = bySlug.get(row.slug);
    if (!film) return [];
    const cells: HeatCell[] = words.map((word, i) => {
      const lift = row.lift?.[i];
      const count = row.count?.[i] ?? 0;
      if (lift === null || lift === undefined) {
        return { level: null };
      }
      return {
        level: liftLevel(lift),
        text: lift >= 2 ? `${lift.toFixed(1)}×` : undefined,
        title: `${filmName(film)}, ${word}: ${lift.toFixed(
          1
        )}× (${count} photos)`,
        href: galleryHref(film, word),
      };
    });
    return [
      {
        label: (
          <Link href={`/films/${film.slug}`} prefetch={false}>
            {filmName(film)}
          </Link>
        ),
        cells,
      },
    ];
  });

  return (
    <section className={shared.section}>
      <h2 className={shared.heading}>WHAT EACH FILM GETS POINTED AT</h2>
      <p className={shared.lead}>
        How much more often a subject shows up on a film than across all photos.
        Each cell opens those photos.
      </p>
      <HeatGrid columns={words} rows={rows} ariaLabel="subjects by film" />
      <p className={shared.note}>Empty cells have fewer than 15 photos.</p>
    </section>
  );
}
