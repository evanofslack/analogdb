import {
  AdminCoverage,
  AdminPost,
  AdminQuality,
  AdminQueue,
  MissingField,
  missingFields,
  MissingPosts as MissingPostsPage,
  UnmatchedCamera,
  UnmatchedFilm,
} from "@lib/adminClient";
import { formatNumber, formatPercent } from "@lib/format";
import { Progress } from "@mantine/core";
import Link from "next/link";
import styles from "./adminPanel.module.css";
import CatalogButton from "./catalogButton";
import EditPostButton from "./editPostButton";
import Pager from "./pager";
import Section from "./section";
import Segmented from "./segmented";
import TabError, { LoadError } from "./tabError";

const coverageRows = [
  { key: "camera", label: "Camera" },
  { key: "film", label: "Film" },
  { key: "description", label: "Description" },
  { key: "keywords", label: "Keywords" },
  { key: "colors", label: "Colors" },
  { key: "focal_length", label: "Focal length" },
  { key: "aperture", label: "Aperture" },
] as const;

interface CoverageProps {
  coverage: AdminCoverage;
}

function Coverage({ coverage }: CoverageProps) {
  return (
    <div className={styles.coverage}>
      {coverageRows.map((row) => {
        const value = coverage[row.key];
        const pct = coverage.total ? (value / coverage.total) * 100 : 0;
        return (
          <div key={row.key} className={styles.coverageRow}>
            <span>{row.label}</span>
            <Progress value={pct} size={4} radius={4} color="gray" />
            <span className={styles.num}>
              {formatPercent(value, coverage.total)}
            </span>
          </div>
        );
      })}
    </div>
  );
}

interface UnmatchedCamerasProps {
  cameras: UnmatchedCamera[];
}

function UnmatchedCameras({ cameras }: UnmatchedCamerasProps) {
  if (cameras.length === 0) {
    return <p className={styles.empty}>All cameras are in the catalog</p>;
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
              <td className={styles.nowrap}>
                <Link
                  href={`/post/${c.sample_post_id}`}
                  className={styles.link}
                >
                  #{c.sample_post_id}
                </Link>
              </td>
              <td className={styles.num}>
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

interface UnmatchedFilmsProps {
  films: UnmatchedFilm[];
}

function UnmatchedFilms({ films }: UnmatchedFilmsProps) {
  if (films.length === 0) {
    return <p className={styles.empty}>All films are in the catalog</p>;
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
              <td className={styles.nowrap}>
                <Link
                  href={`/post/${f.sample_post_id}`}
                  className={styles.link}
                >
                  #{f.sample_post_id}
                </Link>
              </td>
              <td className={styles.num}>
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

function describePost(post: AdminPost): string {
  const camera = [post.camera_make, post.camera_model]
    .filter(Boolean)
    .join(" ");
  const film = [post.film_make, post.film_type].filter(Boolean).join(" ");
  return [camera, film].filter(Boolean).join(" · ") || "no camera or film";
}

interface MissingPostsProps {
  missing?: MissingPostsPage;
  error?: LoadError;
  field: MissingField;
  firstPage: boolean;
}

function MissingPosts({ missing, error, field, firstPage }: MissingPostsProps) {
  return (
    <>
      <div className={styles.controls}>
        <Segmented
          active={field}
          items={missingFields.map((f) => ({
            key: f,
            href: `/admin?tab=catalog&field=${f}`,
            label: f,
          }))}
        />
      </div>
      {error ? (
        <TabError error={error} />
      ) : missing.posts.length === 0 ? (
        <p className={styles.empty}>No posts missing {field}</p>
      ) : (
        <div className={styles.rows}>
          {missing.posts.map((post) => (
            <div key={post.id} className={styles.row}>
              <Link href={`/post/${post.id}`}>
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  className={styles.thumb}
                  src={post.low_url}
                  alt={post.title}
                  loading="lazy"
                />
              </Link>
              <div className={styles.rowText}>
                <div>
                  <Link href={`/post/${post.id}`} className={styles.link}>
                    <span className={styles.muted}>#{post.id}</span>{" "}
                    {post.title}
                  </Link>
                </div>
                <div className={styles.meta}>{describePost(post)}</div>
              </div>
              <EditPostButton id={post.id} />
            </div>
          ))}
        </div>
      )}
      <Pager
        newest={firstPage ? null : `/admin?tab=catalog&field=${field}`}
        older={
          missing?.next_before_id
            ? `/admin?tab=catalog&field=${field}&before_id=${missing.next_before_id}`
            : null
        }
      />
    </>
  );
}

interface CatalogTabProps {
  quality: AdminQuality;
  queue?: AdminQueue;
  missing?: MissingPostsPage;
  missingError?: LoadError;
  field: MissingField;
  firstPage: boolean;
}

export default function CatalogTab({
  quality,
  queue,
  missing,
  missingError,
  field,
  firstPage,
}: CatalogTabProps) {
  return (
    <>
      <Section
        title="Cameras not in catalog"
        count={queue?.unmatched_cameras ?? quality.unmatched_cameras.length}
      >
        <UnmatchedCameras cameras={quality.unmatched_cameras} />
      </Section>

      <Section
        title="Films not in catalog"
        count={queue?.unmatched_films ?? quality.unmatched_films.length}
      >
        <UnmatchedFilms films={quality.unmatched_films} />
      </Section>

      <Section title="Coverage" count={quality.coverage.total}>
        <Coverage coverage={quality.coverage} />
      </Section>

      <Section title="Missing a field">
        <MissingPosts
          missing={missing}
          error={missingError}
          field={field}
          firstPage={firstPage}
        />
      </Section>
    </>
  );
}
