"use client";

import {
  filmName,
  galleryHref,
  isColorFilm,
  lookSectionId,
  popularity,
  share,
  StatsFilm,
  statsParsers,
  styleLabels,
} from "@lib/stats";
import { Select } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import Link from "next/link";
import { useQueryStates } from "nuqs";
import { Fragment, ReactNode, useState } from "react";
import HueWheel from "./hueWheel";
import styles from "./lookSection.module.css";
import shared from "./stats.module.css";
import StatsPhoto from "./statsPhoto";

const shownFilms = 16;
const styleRange = 1.2;

function WarmCool({ film }: { film: StatsFilm }) {
  const warm = film.bands?.warm ?? 0;
  const cool = film.bands?.cool ?? 0;
  const warmShare = warm + cool > 0 ? warm / (warm + cool) : 0.5;
  return (
    <div
      className={styles.warmCool}
      title={`warm ${share(warm)}, cool ${share(cool)}`}
    >
      <span className={styles.warm} style={{ width: `${warmShare * 100}%` }} />
      <span className={styles.cool} />
    </div>
  );
}

function Palette({ colors }: { colors: string[] }) {
  return (
    <div className={styles.palette}>
      {colors.map((color, i) => (
        <span key={i} style={{ background: color }} />
      ))}
    </div>
  );
}

function StyleBars({ film, only }: { film: StatsFilm; only?: string[] }) {
  const rows = styleLabels.filter((s) => !only || only.includes(s.key));
  return (
    <div className={styles.styleBars}>
      {rows.map(({ key, label }) => {
        const value = film.styles?.[key] ?? 0;
        const clamped = Math.max(-styleRange, Math.min(styleRange, value));
        const width = (Math.abs(clamped) / styleRange) * 50;
        return (
          <div
            key={key}
            className={styles.styleRow}
            title={`${label} ${value.toFixed(2)}`}
          >
            <span className={styles.styleLabel}>{label}</span>
            <span className={styles.styleTrack}>
              <span
                className={styles.styleBar}
                style={
                  clamped >= 0
                    ? { left: "50%", width: `${width}%` }
                    : { left: `${50 - width}%`, width: `${width}%` }
                }
              />
            </span>
          </div>
        );
      })}
    </div>
  );
}

function Bands({ film }: { film: StatsFilm }) {
  const bands = film.bands;
  if (!bands) return null;
  const rows = [
    { label: "warm", value: bands.warm ?? 0, color: "oklch(0.72 0.12 70)" },
    { label: "green", value: bands.green ?? 0, color: "oklch(0.72 0.12 145)" },
    { label: "cool", value: bands.cool ?? 0, color: "oklch(0.72 0.12 235)" },
    {
      label: "magenta",
      value: bands.magenta ?? 0,
      color: "oklch(0.72 0.12 330)",
    },
  ];
  return (
    <div className={styles.bands}>
      {rows.map((row) => (
        <div key={row.label} className={styles.bandRow}>
          <span className={styles.bandLabel}>{row.label}</span>
          <span className={styles.bandTrack}>
            <span
              style={{ width: `${row.value * 100}%`, background: row.color }}
            />
          </span>
          <span className={styles.bandValue}>{share(row.value)}</span>
        </div>
      ))}
    </div>
  );
}

interface PanelProps {
  film: StatsFilm;
  films: StatsFilm[];
  onSelect: (slug: string) => void;
  onClose: () => void;
}

function LookPanel({ film, films, onSelect, onClose }: PanelProps) {
  const [compareSlug, setCompareSlug] = useState<string | null>(null);
  const color = isColorFilm(film);
  const others = films.filter((f) => f.slug !== film.slug && isColorFilm(f));
  const compare = others.find((f) => f.slug === compareSlug);
  const bySlug = new Map(films.map((f) => [f.slug, f]));

  return (
    <div className={styles.panel}>
      <div className={styles.panelHead}>
        <Link
          href={`/films/${film.slug}`}
          prefetch={false}
          className={styles.panelTitle}
        >
          {filmName(film)}
        </Link>
        <span className={styles.panelMeta}>
          {(film.postCount ?? 0).toLocaleString("en-US")} photos · popularity{" "}
          {popularity(film.popularity)}
        </span>
        <button
          type="button"
          className={styles.close}
          onClick={onClose}
          aria-label="close"
        >
          ×
        </button>
      </div>
      <div className={styles.panelBody}>
        {color && (
          <div className={styles.panelLook}>
            <HueWheel
              hue={film.hue ?? []}
              compare={compare?.hue}
              size={180}
              label={`hue wheel of ${filmName(film)}`}
            />
            <div className={styles.panelLookSide}>
              <Bands film={film} />
              <Select
                size="xs"
                placeholder="compare with…"
                data={others.map((f) => ({
                  value: f.slug ?? "",
                  label: filmName(f),
                }))}
                value={compareSlug}
                onChange={setCompareSlug}
                clearable
                searchable
                className={styles.compare}
                aria-label="compare with another film"
              />
            </div>
          </div>
        )}
        <div className={styles.panelSide}>
          <div className={shared.subheading}>style</div>
          <StyleBars film={film} />
          <div className={shared.subheading}>looks most like</div>
          <ul className={styles.similar}>
            {(film.similar ?? []).map((s) => {
              const other = bySlug.get(s.slug);
              return (
                <li key={s.slug}>
                  <button type="button" onClick={() => onSelect(s.slug ?? "")}>
                    {other ? filmName(other) : s.slug}
                  </button>
                  <span>{(s.similarity ?? 0).toFixed(2)}</span>
                </li>
              );
            })}
          </ul>
        </div>
      </div>
      <div className={styles.typical}>
        {(film.typical ?? []).map((post) => (
          <StatsPhoto
            key={post.id}
            post={post}
            sizes="(max-width: 720px) 33vw, 160px"
          />
        ))}
      </div>
      <Link href={galleryHref(film)} prefetch={false} className={shared.more}>
        all photos →
      </Link>
    </div>
  );
}

function FilmCard({
  film,
  open,
  onToggle,
}: {
  film: StatsFilm;
  open: boolean;
  onToggle: () => void;
}) {
  return (
    <div className={`${styles.card} ${open ? styles.cardOpen : ""}`}>
      <button
        type="button"
        className={styles.wheelButton}
        onClick={onToggle}
        aria-expanded={open}
        aria-label={`show the look of ${filmName(film)}`}
      >
        <HueWheel
          hue={film.hue ?? []}
          size={120}
          label={`hue wheel of ${filmName(film)}`}
        />
      </button>
      <Link
        href={`/films/${film.slug}`}
        prefetch={false}
        className={styles.cardName}
      >
        {filmName(film)}
      </Link>
      <Palette colors={film.palette ?? []} />
      <WarmCool film={film} />
    </div>
  );
}

// rowsWithPanel places the open film's panel under the row that holds it
function rowsWithPanel(
  items: StatsFilm[],
  columns: number,
  openSlug: string | null,
  render: (film: StatsFilm) => ReactNode,
  panel: ReactNode
): ReactNode[] {
  const out: ReactNode[] = [];
  const openIndex = items.findIndex((f) => f.slug === openSlug);
  items.forEach((film, i) => {
    out.push(<Fragment key={film.slug}>{render(film)}</Fragment>);
    const rowEnd = (i + 1) % columns === 0 || i === items.length - 1;
    if (
      rowEnd &&
      openIndex >= 0 &&
      Math.floor(openIndex / columns) === Math.floor(i / columns)
    ) {
      out.push(
        <div key="panel" className={styles.panelSlot}>
          {panel}
        </div>
      );
    }
  });
  return out;
}

export default function LookSection({ films }: { films: StatsFilm[] }) {
  const [{ film: openSlug }, setParams] = useQueryStates(statsParsers);
  const [showAll, setShowAll] = useState(false);
  const phone = useMediaQuery("(max-width: 720px)");
  const columns = phone ? 2 : 4;

  const color = films.filter(isColorFilm);
  const bw = films.filter((f) => !isColorFilm(f));
  const open = films.find((f) => f.slug === openSlug) ?? null;
  const openHidden = open && color.indexOf(open) >= shownFilms;
  const visible = showAll || openHidden ? color : color.slice(0, shownFilms);

  const select = (slug: string | null) => setParams({ film: slug });
  const toggle = (slug: string | undefined) =>
    select(slug && slug !== openSlug ? slug : null);

  const panel = open && (
    <LookPanel
      key={open.slug}
      film={open}
      films={films}
      onSelect={(slug) => select(slug)}
      onClose={() => select(null)}
    />
  );

  return (
    <section id={lookSectionId} className={shared.section}>
      <h2 className={shared.heading}>THE LOOK OF EACH FILM</h2>
      <p className={shared.lead}>
        Every color stock measured from its photos: where its colors fall on the
        wheel and how warm or cool it runs. Pick one to see more.
      </p>
      <div className={styles.grid}>
        {rowsWithPanel(
          visible,
          columns,
          open && isColorFilm(open) ? openSlug : null,
          (film) => (
            <FilmCard
              film={film}
              open={film.slug === openSlug}
              onToggle={() => toggle(film.slug)}
            />
          ),
          panel
        )}
      </div>
      {color.length > shownFilms && !openHidden && (
        <button
          type="button"
          className={shared.more}
          onClick={() => setShowAll(!showAll)}
        >
          {showAll ? "fewer films ↑" : `all ${color.length} films →`}
        </button>
      )}

      <div className={shared.subheading}>black and white</div>
      <p className={styles.bwLead}>
        Black and white stocks look nearly identical to each other in what
        they&apos;re used for. Grain and sharpness are where they differ.
      </p>
      <div className={styles.bwList}>
        <div className={styles.bwHead} aria-hidden>
          <span />
          <span>grain</span>
          <span>crisp</span>
          <span>popularity</span>
        </div>
        {bw.map((film) => (
          <Fragment key={film.slug}>
            <button
              type="button"
              className={`${styles.bwRow} ${
                film.slug === openSlug ? styles.bwOpen : ""
              }`}
              onClick={() => toggle(film.slug)}
              aria-expanded={film.slug === openSlug}
            >
              <span className={styles.bwName}>{filmName(film)}</span>
              <StyleBars film={film} only={["grain"]} />
              <StyleBars film={film} only={["crisp"]} />
              <span className={styles.bwPop}>
                {popularity(film.popularity)}
              </span>
            </button>
            {film.slug === openSlug && panel}
          </Fragment>
        ))}
      </div>
    </section>
  );
}
