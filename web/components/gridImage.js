import Image from "next/image";
import Link from "next/link";

export default function GridImage(props) {
    let post = props.post
    let image = post.images[2]
    let placeholder = post.images[0]

    return (
        <Link href={`/post/${post.id}`} passHref={true}>
            <div>
                <Image
                    src={image.url}
                    width={image.width}
                    height={image.height}
                    alt={`Image ${post.id} by ${post.author}`}
                    quality={100}
                    layout="responsive"
                    placeholder="blur"
                    blurDataURL={placeholder.url} // low res image
                    />
              </div>
        </Link>
    );
}
