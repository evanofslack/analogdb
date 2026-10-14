import type { Gear } from "@lib/display";
import { smallImage } from "@lib/images";
import { postAlt } from "@lib/seo";
import { IconCamera, IconMovie } from "@tabler/icons-react";
import { AnalogdbPost } from "analogdb-generated";
import Image from "next/image";
import Link from "next/link";
import styles from "./gridImage.module.css";

interface GridImageProps {
  post: AnalogdbPost;
  sizes: string;
  gear: Gear;
  priority?: boolean;
  onClick?: () => void;
}

const joined = (...parts: (string | undefined)[]) =>
  parts.filter(Boolean).join(" ");

const hasGear = (post: AnalogdbPost) =>
  Boolean(
    post.cameraMake || post.cameraModel || post.filmMake || post.filmType
  );

function GearLine({ post }: { post: AnalogdbPost }) {
  const camera = joined(post.cameraMake, post.cameraModel);
  const film = joined(post.filmMake, post.filmType);
  if (!camera && !film) return null;
  return (
    <span className={styles.gearLine}>
      {camera && (
        <span className={styles.gearItem}>
          <IconCamera size={13} stroke={1.5} aria-hidden="true" />
          <span className={styles.gearText}>{camera}</span>
        </span>
      )}
      {film && (
        <span className={styles.gearItem}>
          <IconMovie size={13} stroke={1.5} aria-hidden="true" />
          <span className={styles.gearText}>{film}</span>
        </span>
      )}
    </span>
  );
}

export default function GridImage(props: GridImageProps) {
  let post = props.post;
  if (post == null) {
    return;
  }

  let image = smallImage(post.images);

  return (
    <Link
      href={`/post/${post.id}`}
      passHref={true}
      prefetch={false}
      onClick={props.onClick}
      className={styles.tile}
      // iOS Safari only applies :active when a touch listener is attached
      onTouchStart={() => {}}
      data-post-id={post.id}
    >
      <div className={styles.frame}>
        <Image
          src={image.url}
          width={image.width}
          height={image.height}
          alt={postAlt(post)}
          sizes={props.sizes}
          priority={props.priority}
          fetchPriority={props.priority ? "high" : undefined}
          style={{
            objectFit: "cover",
            width: "100%",
            height: "auto",
            backgroundColor: post.colors?.[0]?.hex,
          }}
        />
        {props.gear === "hover" && hasGear(post) && (
          <div className={styles.overlay}>
            <GearLine post={post} />
          </div>
        )}
      </div>
      {props.gear === "always" && (
        <div className={styles.caption}>
          <GearLine post={post} />
        </div>
      )}
    </Link>
  );
}
