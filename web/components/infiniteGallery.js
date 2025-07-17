"use client";

import Grid from "@components/grid";
import { baseURL } from "@lib/constants";
import { Loader, Skeleton } from "@mantine/core";
import { useEffect, useState } from "react";
import InfiniteScroll from "react-infinite-scroll-component";
import styles from "./infiniteGallery.module.css";

export default function InfiniteGallery(props) {
  const { response, _ } = props;

  const [posts, setPosts] = useState([]);
  const [nextPageRoute, setNextPageRoute] = useState(null);
  const [hasMore, setHasMore] = useState(false);
  const [totalPosts, setTotalPosts] = useState(0);
  const [isLocalLoading, setIsLocalLoading] = useState(true);

  // Update state when response changes
  useEffect(() => {
    if (response && response.posts) {
      setPosts(response.posts);
      setNextPageRoute(
        response.meta?.nextPageUrl ? baseURL + response.meta.nextPageUrl : null
      );
      setHasMore(!!response.meta?.nextPageId);
      setTotalPosts(response.meta?.totalPosts || 0);
      setIsLocalLoading(false);
    }
  }, [response]);

  // Show skeleton loaders on initial load
  if (isLocalLoading && posts.length === 0) {
    return (
      <div className={styles.skeletonContainer}>
        <div className={styles.skeletonGrid}>
          {[...Array(25)].map((_, index) => (
            <Skeleton key={index} height={300} radius="md" animate={true} />
          ))}
        </div>
      </div>
    );
  }

  // Handle null/undefined response
  if (
    !isLocalLoading &&
    response &&
    response.posts &&
    response.posts.length === 0
  ) {
    return (
      <div className={styles.noResultsContainer}>
        <h3 className={styles.noResults}>no posts found :(</h3>
      </div>
    );
  }

  // Fetch next page of results for infinite scroll
  const fetchMore = () => {
    if (!nextPageRoute) return;

    fetch(nextPageRoute)
      .then((res) => res.json())
      .then((response) => {
        if (response.meta.nextPageId == "") {
          setHasMore(false);
        } else {
          setHasMore(true);
          setNextPageRoute(baseURL + response.meta.nextPageUrl);
        }
        setPosts(posts.concat(response.posts));
      });
  };

  const loader = () => (
    <h4 className={styles.loading}>
      <Loader color="gray" variant="dots" />
    </h4>
  );

  return (
    <div>
      {totalPosts != 0 && (
        <div>
          <InfiniteScroll
            dataLength={posts.length}
            next={fetchMore}
            hasMore={hasMore}
            loader={loader()}
            endMessage={
              <h3 className={styles.end}>
                thats all folks, go take some pictures...
              </h3>
            }
            style={{ overflowY: "hidden" }}
          >
            <Grid posts={posts} />
            <span />
          </InfiniteScroll>
        </div>
      )}
      {!isLocalLoading && totalPosts == 0 && (
        <div className={styles.noResultsContainer}>
          <h3 className={styles.noResults}> no posts found :( </h3>
        </div>
      )}
    </div>
  );
}
