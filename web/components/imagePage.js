import Image from "next/image";
import ImageTag from "../components/imageTag";
import styles from "./imagePage.module.css";

export default function ImagePage(props) {
  let post = props.post;
  let image = post.images[3];
  let placeholder = post.images[0];
  return (
    <div className={styles.fullscreen}>
      <ImageTag post={post} />
      <div className={styles.imageContainer}>
        <Image
          fill
          src={image.url}
          alt={`Image ${post.id} by ${post.author}`}
          sizes="100vw"
          style={{ objectFit: "cover" }}
          quality={100}
        />
      </div>
    </div>
  );
}

// <Image
// src={image.url}
// alt={`Image ${post.id} by ${post.author}`}
// width={image.width}
// height={image.height}
// quality={100}
// layout="fill"
// objectFit="contain"
// priority={true}
