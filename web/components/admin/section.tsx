import { ReactNode } from "react";
import styles from "./adminPanel.module.css";

interface SectionProps {
  title?: ReactNode;
  count?: number | null;
  right?: ReactNode;
  level?: "h2" | "h3";
  panel?: boolean;
  className?: string;
  children: ReactNode;
}

export default function Section({
  title,
  count,
  right,
  level = "h2",
  panel,
  className,
  children,
}: SectionProps) {
  const Heading = level;
  const classes = [styles.section, panel && styles.panel, className]
    .filter(Boolean)
    .join(" ");
  return (
    <section className={classes}>
      {(title || right) && (
        <div className={styles.sectionHead}>
          {title && (
            <Heading className={styles.sectionTitle}>
              {title}
              {count !== undefined && count !== null && (
                <span className={styles.sectionCount}>
                  {count.toLocaleString("en-US")}
                </span>
              )}
            </Heading>
          )}
          {right && <div className={styles.sectionRight}>{right}</div>}
        </div>
      )}
      {children}
    </section>
  );
}
