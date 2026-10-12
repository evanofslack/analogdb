import { smallImage } from "@lib/images";
import { postAlt } from "@lib/seo";
import { AnalogdbPost } from "analogdb-generated";
import Image from "next/image";
import Link from "next/link";

interface GridImageProps {
  post: AnalogdbPost;
  priority?: boolean;
  onClick?: () => void;
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
    >
      <div>
        <Image
          src={image.url}
          width={image.width}
          height={image.height}
          alt={postAlt(post)}
          sizes="(max-width: 720px) 50vw, (max-width: 1440px) 33vw, 25vw"
          priority={props.priority}
          fetchPriority={props.priority ? "high" : undefined}
          style={{
            objectFit: "cover",
            width: "100%",
            height: "auto",
            backgroundColor: post.colors?.[0]?.hex,
          }}
        />
      </div>
    </Link>
  );
}
