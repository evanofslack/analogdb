import { popularity, Stats } from "@lib/stats";
import styles from "./popularitySection.module.css";
import shared from "./stats.module.css";
import StatsPhoto from "./statsPhoto";

function Dot({ value, hollow }: { value: number; hollow?: boolean }) {
  return (
    <span
      className={hollow ? styles.dotHollow : styles.dot}
      style={{ left: `${value}%` }}
      title={`${hollow ? "black and white" : "color"} ${popularity(value)}`}
    />
  );
}

function MiniBars({ values, labels }: { values: number[]; labels?: string[] }) {
  const max = Math.max(...values, 1);
  return (
    <div className={styles.miniBars}>
      {values.map((v, i) => (
        <div
          key={i}
          className={styles.miniBar}
          title={`${labels?.[i] ?? i + 1}: ${popularity(v)}`}
        >
          <span style={{ height: `${(v / max) * 100}%` }} />
          {labels && <small>{labels[i]}</small>}
        </div>
      ))}
    </div>
  );
}

export default function PopularitySection({ stats }: { stats: Stats }) {
  const themes = stats.themes ?? [];
  const pop = stats.popularity ?? {};
  const borderTheme = themes.find((t) => t.key === pop.border?.theme);
  const borderPost =
    borderTheme?.posts?.find((p) => !p.nsfw) ?? borderTheme?.posts?.[0];
  const apertures = pop.aperture ?? [];
  const dotValue = (t: (typeof themes)[number]) =>
    t.color?.popularity ?? t.popularity ?? 0;
  const ranked = [...themes].sort((a, b) => dotValue(b) - dotValue(a));
  const colorAhead = (pop.color ?? 0) > (pop.bw ?? 0);
  const borderBelow = (pop.border?.popularity ?? 50) < 45;

  return (
    <section className={shared.section}>
      <h2 className={shared.heading}>WHAT DOES WELL</h2>
      <p className={shared.lead}>
        Popularity is a photo&apos;s score rank within its month, from 0 to 100,
        so 50 is average. These are patterns in this collection, not rules.
      </p>

      <div className={styles.callouts}>
        <div className={styles.card}>
          <div className={styles.cardTitle}>color vs black and white</div>
          <div className={styles.cardValue}>
            {popularity(pop.color)} <span>vs</span> {popularity(pop.bw)}
          </div>
          <p>
            {colorAhead
              ? "Color photos rank higher than black and white ones, and the gap holds within most themes below."
              : "Color and black and white photos rank about the same."}
          </p>
        </div>
        {pop.border && (
          <div className={styles.card}>
            <div className={styles.cardTitle}>showing the film border</div>
            <div className={styles.cardRow}>
              <div className={styles.cardValue}>
                {popularity(pop.border.popularity)}
              </div>
              {borderPost && (
                <StatsPhoto
                  post={borderPost}
                  sizes="64px"
                  className={styles.cardPhoto}
                />
              )}
            </div>
            <p>
              Photos that show the sprocket holes or edge print
              {borderBelow
                ? " rank well below average."
                : " rank about average."}
            </p>
          </div>
        )}
        {(pop.saturation ?? []).length > 0 && (
          <div className={styles.card}>
            <div className={styles.cardTitle}>saturation hardly matters</div>
            <MiniBars values={pop.saturation ?? []} />
            <p>
              Color photos from least to most saturated, in ten equal groups.
            </p>
          </div>
        )}
        {apertures.length > 0 && (
          <div className={styles.card}>
            <div className={styles.cardTitle}>
              wide open isn&apos;t rewarded
            </div>
            <MiniBars
              values={apertures.map((a) => a.popularity ?? 0)}
              labels={apertures.map((a) => (a.value ?? "").replace("f/", ""))}
            />
            <p>Popularity by aperture, f-number left to right.</p>
          </div>
        )}
      </div>

      <div className={shared.subheading}>themes ranked</div>
      <div className={styles.scaleHead} aria-hidden>
        <span />
        <span className={styles.scale}>
          <small style={{ left: "0%" }}>0</small>
          <small style={{ left: "50%" }}>50</small>
          <small style={{ left: "100%" }}>100</small>
        </span>
      </div>
      <ul className={styles.themes}>
        {ranked.map((theme) => {
          const color = dotValue(theme);
          const bw = theme.bw?.popularity;
          return (
            <li key={theme.key} className={styles.theme}>
              <div className={styles.themeLabel}>
                <div className={styles.themePhotos}>
                  {(theme.posts ?? []).slice(0, 2).map((post) => (
                    <StatsPhoto
                      key={post.id}
                      post={post}
                      sizes="40px"
                      className={styles.themePhoto}
                    />
                  ))}
                </div>
                <span className={styles.themeName}>{theme.name}</span>
              </div>
              <span className={styles.track}>
                <span className={styles.middle} />
                {bw !== undefined && bw !== null && (
                  <span
                    className={styles.gap}
                    style={{
                      left: `${Math.min(color, bw)}%`,
                      width: `${Math.abs(color - bw)}%`,
                    }}
                  />
                )}
                <Dot value={color} />
                {bw !== undefined && bw !== null && <Dot value={bw} hollow />}
              </span>
            </li>
          );
        })}
      </ul>
      <p className={shared.note}>
        Filled dots are color photos, hollow dots black and white photos of the
        same theme where there are at least 30.
      </p>
    </section>
  );
}
