import Link from "next/link";
import styles from "./keywords.module.css";

export default function Keywords({ keywords, maxKeywords = 15 }) {
  if (!keywords || keywords.length === 0) {
    return null;
  }

  return (
    <div className={styles.containerKeywords}>
      {keywords.slice(0, maxKeywords).map((item) => (
        <Link
          href={`/?text=${encodeURIComponent(item.word)}`}
          prefetch={false}
          className={styles.keyword}
          key={item.word}
        >
          {item.word}
        </Link>
      ))}
    </div>
  );
}
