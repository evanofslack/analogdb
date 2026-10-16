import Link from "next/link";
import { ReactNode } from "react";
import styles from "./heatGrid.module.css";

export interface HeatCell {
  // 0 to 1, how strong the cell's color is, null for an empty cell
  level: number | null;
  text?: string;
  title?: string;
  href?: string;
}

interface HeatGridProps {
  columns: string[];
  rows: { label: ReactNode; cells: HeatCell[] }[];
  // narrow cells with labels only on some columns, for long grids
  compact?: boolean;
  columnLabel?: (column: string, index: number) => string;
  ariaLabel: string;
}

export default function HeatGrid({
  columns,
  rows,
  compact,
  columnLabel,
  ariaLabel,
}: HeatGridProps) {
  const template = `${compact ? "2.6rem" : "var(--heat-label)"} repeat(${
    columns.length
  }, minmax(${compact ? "0.9rem" : "2.2rem"}, 1fr))`;
  return (
    <div className={styles.scroll}>
      <div
        className={`${styles.grid} ${compact ? styles.compact : ""}`}
        style={{ gridTemplateColumns: template }}
        role="table"
        aria-label={ariaLabel}
      >
        <div
          className={`${styles.corner} ${styles.sticky}`}
          role="columnheader"
        />
        {columns.map((column, i) => (
          <div key={column} className={styles.column} role="columnheader">
            {columnLabel ? columnLabel(column, i) : column}
          </div>
        ))}
        {rows.map((row, r) => (
          <div key={r} className={styles.row} role="row">
            <div
              className={`${styles.rowLabel} ${styles.sticky}`}
              role="rowheader"
            >
              {row.label}
            </div>
            {row.cells.map((cell, c) => {
              const style =
                cell.level === null
                  ? undefined
                  : {
                      background: `color-mix(in oklab, var(--adb-heat) ${Math.round(
                        cell.level * 100
                      )}%, var(--adb-surface))`,
                    };
              const strong = cell.level !== null && cell.level > 0.55;
              const className = `${styles.cell} ${strong ? styles.strong : ""}`;
              return cell.href ? (
                <Link
                  key={c}
                  href={cell.href}
                  prefetch={false}
                  className={className}
                  style={style}
                  title={cell.title}
                  role="cell"
                >
                  {cell.text}
                </Link>
              ) : (
                <div
                  key={c}
                  className={className}
                  style={style}
                  title={cell.title}
                  role="cell"
                >
                  {cell.text}
                </div>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
}
