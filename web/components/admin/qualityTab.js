import { missingFields } from "@lib/adminClient";
import { formatNumber, formatPercent } from "@lib/format";
import { Button, Progress } from "@mantine/core";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import CatalogButton from "./catalogButton";
import EditPostButton from "./editPostButton";
import TabError from "./tabError";

const coverageRows = [
  { key: "camera", label: "Camera" },
  { key: "film", label: "Film" },
  { key: "description", label: "Description" },
  { key: "keywords", label: "Keywords" },
  { key: "colors", label: "Colors" },
  { key: "focal_length", label: "Focal length" },
  { key: "aperture", label: "Aperture" },
];

function Coverage({ coverage }) {
  return (
    <div className={styles.coverage}>
      {coverageRows.map((row) => {
        const value = coverage[row.key];
        const pct = coverage.total ? (value / coverage.total) * 100 : 0;
        return (
          <div key={row.key} className={styles.coverageRow}>
            <span>{row.label}</span>
            <Progress value={pct} size="lg" radius="sm" />
            <span className={styles.num}>
              {formatPercent(value, coverage.total)}
            </span>
          </div>
        );
      })}
      <p className={styles.note}>
        Out of {formatNumber(coverage.total)} posts.
      </p>
    </div>
  );
}

function UnmatchedCameras({ cameras }) {
  if (cameras.length === 0) {
    return (
      <p className={styles.note}>Every camera on a post is in the catalog.</p>
    );
  }
  return (
    <div className={styles.tableWrap}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th>Make</th>
            <th>Model</th>
            <th className={styles.num}>Posts</th>
            <th>Example</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {cameras.map((c) => (
            <tr key={`${c.camera_make}|${c.camera_model}`}>
              <td>{c.camera_make}</td>
              <td>{c.camera_model}</td>
              <td className={styles.num}>{formatNumber(c.post_count)}</td>
              <td>
                <Link
                  href={`/post/${c.sample_post_id}`}
                  className={styles.link}
                >
                  #{c.sample_post_id}
                </Link>
              </td>
              <td>
                <CatalogButton
                  kind="camera"
                  initial={{
                    make: c.camera_make,
                    model: c.camera_model,
                    description: "",
                  }}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function UnmatchedFilms({ films }) {
  if (films.length === 0) {
    return (
      <p className={styles.note}>Every film on a post is in the catalog.</p>
    );
  }
  return (
    <div className={styles.tableWrap}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th>Make</th>
            <th>Type</th>
            <th className={styles.num}>Speed</th>
            <th className={styles.num}>Posts</th>
            <th>Example</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {films.map((f) => (
            <tr key={`${f.film_make}|${f.film_type}`}>
              <td>{f.film_make}</td>
              <td>{f.film_type}</td>
              <td className={styles.num}>{f.film_speed ?? "—"}</td>
              <td className={styles.num}>{formatNumber(f.post_count)}</td>
              <td>
                <Link
                  href={`/post/${f.sample_post_id}`}
                  className={styles.link}
                >
                  #{f.sample_post_id}
                </Link>
              </td>
              <td>
                <CatalogButton
                  kind="film"
                  initial={{
                    make: f.film_make,
                    type: f.film_type,
                    speed: f.film_speed ?? "",
                    colorType: "color",
                    description: "",
                  }}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function describePost(post) {
  const camera = [post.camera_make, post.camera_model]
    .filter(Boolean)
    .join(" ");
  const film = [post.film_make, post.film_type].filter(Boolean).join(" ");
  return [camera, film].filter(Boolean).join(" · ") || "no camera or film";
}

function MissingPosts({ missing, error, field }) {
  return (
    <>
      <div className={styles.controls}>
        {missingFields.map((f) => (
          <Link
            key={f}
            href={`/admin?tab=quality&field=${f}`}
            className={`${styles.chip} ${f === field ? styles.chipActive : ""}`}
          >
            {f}
          </Link>
        ))}
      </div>
      {error ? (
        <TabError error={error} />
      ) : missing.posts.length === 0 ? (
        <p className={styles.note}>No posts are missing {field}.</p>
      ) : (
        <div className={styles.missingList}>
          {missing.posts.map((post) => (
            <div key={post.id} className={styles.missingItem}>
              <Link href={`/post/${post.id}`}>
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={post.low_url} alt={post.title} loading="lazy" />
              </Link>
              <div className={styles.missingText}>
                <div>
                  <Link href={`/post/${post.id}`} className={styles.link}>
                    #{post.id}
                  </Link>{" "}
                  {post.title}
                </div>
                <div className={styles.note}>{describePost(post)}</div>
              </div>
              <EditPostButton id={post.id} />
            </div>
          ))}
        </div>
      )}
      {missing?.next_before_id && (
        <div className={styles.pager}>
          <Button
            component={Link}
            href={`/admin?tab=quality&field=${field}&before_id=${missing.next_before_id}`}
            variant="default"
          >
            Older
          </Button>
        </div>
      )}
    </>
  );
}

export default function QualityTab({ quality, missing, missingError, field }) {
  return (
    <>
      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Coverage</h2>
        <Coverage coverage={quality.coverage} />
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Cameras not in the catalog</h2>
        <UnmatchedCameras cameras={quality.unmatched_cameras} />
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Films not in the catalog</h2>
        <UnmatchedFilms films={quality.unmatched_films} />
        <p className={styles.note}>
          Films match the catalog on make and type. Speed shows the most common
          value on posts.
        </p>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Posts missing a field</h2>
        <MissingPosts missing={missing} error={missingError} field={field} />
      </section>
    </>
  );
}
