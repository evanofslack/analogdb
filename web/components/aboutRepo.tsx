"use client";

import { RepoData } from "@lib/about";
import { formatAgo, formatNumber } from "@lib/format";
import {
  IconBrandGithub,
  IconGitCommit,
  IconScale,
  IconStar,
} from "@tabler/icons-react";
import { useEffect, useState } from "react";
import styles from "./aboutRepo.module.css";

const REPO_NAME = "evanofslack/analogdb";
const REPO_URL = `https://github.com/${REPO_NAME}`;

export default function AboutRepo({ repo }: { repo: RepoData }) {
  // the time is read after mount so server and client render the same text
  const [updated, setUpdated] = useState<string | null>(null);
  useEffect(() => {
    if (!repo.pushedAt) return;
    const pushed = Math.floor(new Date(repo.pushedAt).getTime() / 1000);
    setUpdated(`updated ${formatAgo(pushed, Date.now())}`);
  }, [repo.pushedAt]);

  return (
    <a className={styles.card} href={REPO_URL}>
      <div className={styles.name}>
        <IconBrandGithub size={20} stroke={1.5} aria-hidden />
        <span>{REPO_NAME}</span>
      </div>
      {repo.description && (
        <p className={styles.description}>{repo.description}</p>
      )}
      {repo.languages.length > 0 && (
        <div className={styles.languages}>
          <div className={styles.bar}>
            {repo.languages.map((language) => (
              <span
                key={language.name}
                style={{
                  width: `${language.percent}%`,
                  backgroundColor: language.color,
                }}
              />
            ))}
          </div>
          <div className={styles.legend}>
            {repo.languages.map((language) => (
              <span key={language.name} className={styles.legendItem}>
                <span
                  className={styles.dot}
                  style={{ backgroundColor: language.color }}
                />
                <span className={styles.legendName}>{language.name}</span>
                {language.percent.toFixed(1)}%
              </span>
            ))}
          </div>
        </div>
      )}
      <div className={styles.stats}>
        <span className={styles.stat}>
          <IconStar size={16} stroke={1.5} aria-hidden />
          {formatNumber(repo.stars)} stars
        </span>
        {repo.license && (
          <span className={styles.stat}>
            <IconScale size={16} stroke={1.5} aria-hidden />
            {repo.license}
          </span>
        )}
        <span className={styles.stat}>
          <IconGitCommit size={16} stroke={1.5} aria-hidden />
          {/* the placeholder holds the width until the time is known */}
          <span className={updated ? undefined : styles.pending}>
            {updated ?? "updated 00d ago"}
          </span>
        </span>
      </div>
    </a>
  );
}
