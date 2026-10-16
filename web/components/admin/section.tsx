import { ReactNode } from "react";
import styles from "./adminPanel.module.css";

interface SectionProps {
  title: ReactNode;
  count?: number | null;
  right?: ReactNode;
  level?: "h2" | "h3";
  className?: string;
  children: ReactNode;
}

export default function Section({
  title,
  count,
  right,
  level = "h2",
  className,
  children,
}: SectionProps) {
  const Heading = level;
  return (
    <section className={`${styles.section} ${className ?? ""}`}>
      <div className={styles.sectionHead}>
        <Heading className={styles.sectionTitle}>
          {title}
          {count !== undefined && count !== null && (
            <span className={styles.sectionCount}>
              {count.toLocaleString("en-US")}
            </span>
          )}
        </Heading>
        {right && <div className={styles.sectionRight}>{right}</div>}
      </div>
      {children}
    </section>
  );
}
