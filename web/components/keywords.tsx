import { keywordHref } from "@lib/keywords";
import { AnalogdbKeyword } from "analogdb-generated";
import Link from "next/link";
import styles from "./keywords.module.css";

interface KeywordsProps {
  keywords: AnalogdbKeyword[];
  maxKeywords?: number;
}

export default function Keywords({
  keywords,
  maxKeywords = 15,
}: KeywordsProps) {
  if (!keywords || keywords.length === 0) {
    return null;
  }

  return (
    <div className={styles.containerKeywords}>
      {keywords.slice(0, maxKeywords).map((item) => (
        <Link
          href={keywordHref(item.word)}
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
