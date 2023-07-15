import styles from "./infiniteGallery.module.css";
import Grid from "../components/grid";
import InfiniteScroll from "react-infinite-scroll-component";
import { useState, useEffect } from "react";
import { baseURL } from "../constants.ts";
import Footer from "./footer";

export default function InfiniteGallery(props) {
  const [posts, setPosts] = useState(props.response.posts);
  const [nextPageURL, setNextPageURL] = useState(
    baseURL + props.response.meta.next_page_url
  );
  const [hasMore, setHasMore] = useState(props.response.meta.next_page_id);
  const [totalPosts, setTotalPosts] = useState(props.response.meta.total_posts);

  // this seems like a hack
  useEffect(() => {
    setPosts(props.response.posts);
    setNextPageURL(baseURL + props.response.meta.next_page_url);
    setHasMore(props.response.meta.next_page_id);
    setTotalPosts(props.response.meta.total_posts);
  }, [props.response]);

  // Fetch next page of results for infinite scroll
  const fetchMore = () => {
    fetch(nextPageURL)
      .then((res) => res.json())
      .then((response) => {
        if (response.meta.next_page_id == "") {
          setHasMore(false);
        } else {
          setHasMore(true);
          setNextPageURL(baseURL + response.meta.next_page_url);
        }
        setPosts(posts.concat(response.posts));
      });
  };

  return (
    <div>
      {totalPosts != 0 && (
        <InfiniteScroll
          dataLength={posts.length}
          next={fetchMore}
          hasMore={hasMore}
          loader={<h4 className={styles.loading}>loading...</h4>}
          endMessage={
            <h3 className={styles.loading}>
              thats all folks, go take some pictures...
            </h3>
          }
          style={{ overflowY: "hidden" }}
        >
          <Grid posts={posts} />
          <span />
        </InfiniteScroll>
      )}
      {totalPosts == 0 && (
        <div className={styles.noResultsContainer}>
          <h3 className={styles.noResults}> no posts found :( </h3>
        </div>
      )}
    </div>
  );
}
