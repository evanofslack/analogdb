import styles from "./imageTag.module.css";
import { baseURL } from "../constants.ts";
import { Tooltip, Badge, Box } from "@mantine/core";
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

  const keywordToColor = (keyword) => {
    const colors = [
      "red",
      "pink",
      "grape",
      "violet",
      "indigo",
      "blue",
      "cyan",
      "teal",
      "green",
      "lime",
      "yellow",
      "orange",
      "dark",
      "gray",
    ];

    switch (keyword[0]) {
      case "e":
        return colors[0];
      case "t":
        return colors[1];
      case "a":
      case "z":
        return colors[2];
      case "o":
      case "j":
        return colors[3];
      case "i":
      case "q":
        return colors[4];
      case "n":
      case "x":
        return colors[5];
      case "s":
      case "k":
        return colors[6];
      case "r":
      case "v":
        return colors[7];
      case "h":
      case "b":
        return colors[8];
      case "d":
      case "p":
        return colors[9];
      case "l":
      case "g":
        return colors[10];
      case "u":
      case "w":
        return colors[11];
      case "c":
      case "y":
        return colors[12];
      case "m":
      case "f":
        return colors[13];
      default:
        return colors[13];
    }
  };

  const keywords =
    Object.hasOwn(post, "keywords") && post.keywords.length > 0
      ? post.keywords
          .map((item) => {
            return item.word;
          })
          .slice(0, 15)
      : [];

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
          <div className={styles.containerColorsAndKeywords}>
            <div className={styles.containerColors}>
              {hex_colors.map((hex) => {
                return (
                  <Tooltip
                    key={hex.id}
                    label={clipboard.copied ? "copied" : hex}
                    position="top"
                    color="gray"
                  >
                    <div
                      key={hex.id}
                      style={color(hex)}
                      className={styles.colorSquare}
                      onClick={() => clipboard.copy(hex)}
                    ></div>
                  </Tooltip>
                );
              })}
            </div>
            <div className={styles.containerKeywords}>
              {keywords.map((word) => {
                return (
                  <div className={styles.keyword} key={word.id}>
                    {word}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
