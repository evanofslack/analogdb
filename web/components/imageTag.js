import styles from "./imageTag.module.css";
import { baseURL } from "../constants.ts";

export default function ImageTag(props) {
  let post = props.post;
  const api_endpoint = baseURL + "/post/";
  const redditUserURL = "https://www.reddit.com/user/";
  const author = post.author.replace("u/", "");

  const date = new Date(post.unix_time * 1000).toLocaleDateString("en-US");

  return (
    <div className={styles.container}>
      <div className={styles.containerMetadata}>
        <a href={post.permalink} className={styles.title}>
          {post.title}
        </a>
        <div className={styles.containerAuthor}>
          <a href={redditUserURL + author} className={styles.author}>
            {author}
          </a>
          <a href={api_endpoint + post.id} className={styles.id}>
            #{post.id}
          </a>
          <div>{date}</div>
        </div>
      </div>
    </div>
  );
}
