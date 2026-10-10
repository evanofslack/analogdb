"use client";

import { FilmSet, shuffle } from "@lib/about";
import Link from "next/link";
import { useEffect, useState } from "react";
import styles from "./aboutFilms.module.css";
import AboutPhoto from "./aboutPhoto";

const SET_SIZE = 4;

export default function AboutFilms({ films }: { films: FilmSet[] }) {
  const [sets, setSets] = useState<FilmSet[] | null>(null);

  // pick after mount so each visit gets different photos
  useEffect(() => {
    setSets(
      films.map((film) => ({
        ...film,
        photos: shuffle(film.photos).slice(0, SET_SIZE),
      }))
    );
  }, [films]);

  if (!sets?.length) return null;

  return (
    <div className={styles.sets}>
      {sets.map((film) => (
        <div key={film.slug} className={styles.set}>
          <div className={styles.tiles}>
            {film.photos.map((photo) => (
              <AboutPhoto
                key={photo.id}
                photo={photo}
                small
                fill
                sizes="(max-width: 720px) 35vw, 120px"
                className={styles.tile}
              />
            ))}
          </div>
          <Link
            href={`/films/${film.slug}`}
            prefetch={false}
            className={styles.label}
          >
            <span className={styles.name}>{film.label}</span>
            {film.postCount > 0 && (
              <span className={styles.count}>
                {film.postCount.toLocaleString()} photos
              </span>
            )}
          </Link>
        </div>
      ))}
    </div>
  );
}
