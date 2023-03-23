import styles from "./imageTag.module.css";
import { baseURL } from "../constants.ts";
import { Tooltip } from "@mantine/core";
import { useClipboard } from "@mantine/hooks";

export default function ImageTag(props) {
  const clipboard = useClipboard({ timeout: 1000 });

  let post = props.post;
  const api_endpoint = baseURL + "/post/";
  const redditUserURL = "https://www.reddit.com/user/";
  const author = post.author.replace("u/", "");

  const date = new Date(post.timestamp * 1000).toLocaleDateString("en-US");

  const c1_hex = post.colors[0].hex;
  const c2_hex = post.colors[1].hex;
  const c3_hex = post.colors[2].hex;
  const c4_hex = post.colors[3].hex;
  const c5_hex = post.colors[4].hex;
  const hex_colors = [c1_hex, c2_hex, c3_hex, c4_hex, c5_hex];

  const color = (hex) => {
    return {
      backgroundColor: hex,
    };
  };

  return (
    <div className={styles.container}>
      <div className={styles.containerMetadata}>
        <a href={post.permalink} className={styles.title}>
          {post.title}
        </a>
        <div className={styles.containerSub}>
          <div className={styles.containerAuthor}>
            <a href={redditUserURL + author} className={styles.author}>
              {author}
            </a>
            <a href={api_endpoint + post.id} className={styles.id}>
              #{post.id}
            </a>
            <div>{date}</div>
          </div>
          <div className={styles.containerColors}>
            {hex_colors.map((hex) => {
              return (
                <Tooltip
                  key={hex.id}
                  label={clipboard.copied ? "copied" : hex}
                  position="bottom"
                  color="gray"
                >
                  <div
                    style={color(hex)}
                    className={styles.colorSquare}
                    onClick={() => clipboard.copy(hex)}
                  ></div>
                </Tooltip>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
