import { Skeleton } from "@mantine/core";
import catalogStyles from "./catalogIndex.module.css";
import GallerySkeleton from "./gallerySkeleton";
import styles from "./searchSkeletons.module.css";
import suggestionStyles from "./searchSuggestions.module.css";

const pillWidths = [72, 96, 64, 110, 84, 70, 102, 88];
const cardCount = 10;

function PillRow() {
  return (
    <div className={styles.pills}>
      {pillWidths.map((width, index) => (
        <Skeleton key={index} width={width} height={34} radius={9} />
      ))}
    </div>
  );
}

function SectionSkeleton({ children }) {
  return (
    <div className={suggestionStyles.section}>
      <Skeleton width={140} height={14} radius="sm" />
      {children}
    </div>
  );
}

export function SearchBarSkeleton() {
  return (
    <>
      <Skeleton className={styles.bar} height={44} />
      <Skeleton className={styles.filter} height={44} />
    </>
  );
}

export function SuggestionsSkeleton() {
  return (
    <div className={suggestionStyles.suggestions}>
      <SectionSkeleton>
        <PillRow />
      </SectionSkeleton>
      <SectionSkeleton>
        <div className={catalogStyles.grid}>
          {[...Array(cardCount)].map((_, index) => (
            <div key={index}>
              <Skeleton className={styles.cover} />
              <Skeleton className={styles.name} height={14} width="60%" />
              <Skeleton className={styles.count} height={12} width="35%" />
            </div>
          ))}
        </div>
      </SectionSkeleton>
    </div>
  );
}

export function ResultsSkeleton({ className }) {
  return (
    <>
      <div className={className}>
        <PillRow />
      </div>
      <GallerySkeleton />
    </>
  );
}
