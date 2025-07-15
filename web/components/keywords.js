"use client";

import { useRouter } from "next/navigation";
import styles from "./keywords.module.css";

export default function Keywords({ keywords, maxKeywords = 15 }) {
  const router = useRouter();

  const handleKeywordClick = (word) => {
    router.push(`/?text=${encodeURIComponent(word)}`);
  };

  if (!keywords || keywords.length === 0) {
    return null;
  }

  return (
    <div className={styles.containerKeywords}>
      {keywords.slice(0, maxKeywords).map((item) => (
        <div className={styles.keyword} key={item.word}>
          <button
            onClick={() => handleKeywordClick(item.word)}
            className={styles.keywordButton}
            type="button"
          >
            {item.word}
          </button>
        </div>
      ))}
    </div>
  );
}
