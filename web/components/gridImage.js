import Image from "next/legacy/image";
import Link from "next/link";

export default function GridImage(props) {
  let post = props.post;
  if (post == null) {
    return;
  }

  let low = post.images[0];
  let medium = post.images[1];
  let placeholder = low;

  // 1st gen low res is too small, use medium res
  // 2nd gen low res is fine
  let image = medium;
  if (low.width >= 720 || low.height >= 720) {
    image = low;
  }

  return (
    <Link href={`/post/${post.id}`} passHref={true} prefetch={false}>
      <div>
        <Image
          src={image.url}
          width={image.width}
          height={image.height}
          alt={`image ${post.id} by ${post.author}`}
          quality={100}
          layout="responsive"
          placeholder="blur"
          blurDataURL={placeholder.url}
        />
      </div>
    </Link>
  );
}
