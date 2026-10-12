"use client";

import GallerySkeleton from "@components/gallerySkeleton";
import Grid from "@components/grid";
import { RateLimitedError } from "@lib/rateLimited";
import { Button, Loader } from "@mantine/core";
import { AnalogdbPost } from "analogdb-generated";
import { useEffect, useMemo, useRef } from "react";
import styles from "./infiniteGallery.module.css";

interface InfiniteGalleryProps {
  initialColumns?: number;
  pages: { posts?: AnalogdbPost[] }[];
  isLoading: boolean;
  isError: boolean;
  error: Error | null;
  isPlaceholderData: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  isFetchNextPageError: boolean;
  fetchNextPage: () => unknown;
  refetch: () => unknown;
  onClick?: (post: AnalogdbPost, position: number) => void;
}

export default function InfiniteGallery({
  initialColumns,
  pages,
  isLoading,
  isError,
  error,
  isPlaceholderData,
  hasNextPage,
  isFetchingNextPage,
  isFetchNextPageError,
  fetchNextPage,
  refetch,
  onClick,
}: InfiniteGalleryProps) {
  const sentinelRef = useRef<HTMLDivElement>(null);
  const postPages = useMemo(
    () => pages.map((page) => page.posts ?? []),
    [pages]
  );
  const posts = postPages.flat();
  const limited = error instanceof RateLimitedError;

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !hasNextPage) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (
          entries[0].isIntersecting &&
          hasNextPage &&
          !isFetchingNextPage &&
          !isPlaceholderData &&
          !isFetchNextPageError
        ) {
          fetchNextPage();
        }
      },
      { rootMargin: "0px 0px 1200px 0px" }
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [
    hasNextPage,
    isFetchingNextPage,
    isPlaceholderData,
    isFetchNextPageError,
    fetchNextPage,
    pages.length,
  ]);

  if (isError && pages.length === 0) {
    return (
      <div className={styles.noResultsContainer}>
        <h3 className={styles.noResults}>
          {limited ? "too many requests, wait a moment" : "couldn't load posts"}
        </h3>
        <Button variant="default" onClick={() => refetch()}>
          retry
        </Button>
      </div>
    );
  }

  if (isLoading) {
    return <GallerySkeleton />;
  }

  if (posts.length === 0) {
    return (
      <div className={styles.noResultsContainer}>
        <h3 className={styles.noResults}>no posts found :(</h3>
      </div>
    );
  }

  return (
    <div>
      {isPlaceholderData && (
        <div className={styles.placeholderLoader}>
          <Loader color="gray" />
        </div>
      )}
      <div className={isPlaceholderData ? styles.dimmed : undefined}>
        <Grid
          pages={postPages}
          initialColumns={initialColumns}
          onClick={onClick}
        />
      </div>
      {hasNextPage && <div ref={sentinelRef} aria-hidden />}
      {isFetchingNextPage && (
        <h4 className={styles.loading}>
          <Loader color="gray" variant="dots" />
        </h4>
      )}
      {isFetchNextPageError && !isFetchingNextPage && (
        <div className={styles.loading}>
          <Button variant="default" onClick={() => fetchNextPage()}>
            {limited
              ? "too many requests · retry"
              : "couldn't load more · retry"}
          </Button>
        </div>
      )}
      {!hasNextPage && !isPlaceholderData && (
        <h3 className={styles.end}>
          thats all folks, go take some pictures...
        </h3>
      )}
    </div>
  );
}
