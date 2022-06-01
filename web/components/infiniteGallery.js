import styles from "../styles/Home.module.css";
import Grid from "../components/grid";
import InfiniteScroll from "react-infinite-scroll-component";
import { useState } from "react";

const baseURL = "https://analogdb.herokuapp.com";

export default function InfiniteGallery(props) {
    const [posts, setPosts] = useState(props.response.posts);
    const [nextPageURL, setNextPageURL] = useState(baseURL + props.response.meta.next_page_url);
    const [hasMore, setHasMore] = useState(props.response.meta.next_page_id);

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
        <InfiniteScroll
            dataLength={posts.length}
            next={fetchMore}
            hasMore={hasMore}
            loader={<h4 className={styles.loading}>Loading...</h4>}
            endMessage={<h4 className={styles.loading}>Go take some pictures...</h4>}
            style={{ overflowY: "hidden" }}
        >
            <Grid posts={posts} />
        </InfiniteScroll>
    );
}
