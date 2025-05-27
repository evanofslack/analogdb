'use client';
import Image from "next/image";
import Link from "next/link";
import ImageTag from "../components/imageTag";
import Footer from "../components/footer";
import styles from "./imagePage.module.css";
import { AiOutlineDownload, AiOutlineArrowsAlt } from "react-icons/ai";
import { ActionIcon, Tooltip } from "@mantine/core";

async function downloadImage(targetImage, name) {
    const image = await fetch(targetImage);
    const imageBlob = await image.blob();
    const imageURL = URL.createObjectURL(imageBlob);
    const link = document.createElement("a");
    link.href = imageURL;
    link.download = `analogdb-${name}`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

export default function ImagePage(props) {
    let post = props.post;
    let similar = props.similar;
    let image = post.images[2];
    let placeholder = post.images[0];

    return (
        <div>
            <div className={styles.fullscreen}>
                <div className={styles.headerIcons}>
                    <Link href={`/`} passHref={true}>
                        <h1 className={styles.title}>Analogdb</h1>
                    </Link>
                </div>
                <div className={styles.imageContainer}>
                    <Image
                        priority
                        style={{ objectFit: "contain" }}
                        fill
                        src={image.url}
                        alt={`image ${post.id} by ${post.author}`}
                        sizes="100vw"
                        quality={100}
                        placeholder="blur"
                        blurDataURL={placeholder.url}
                    />
                </div>
                <div className={styles.footerIcons}>
                    <ActionIcon.Group>
                        <Tooltip label="download" withArrow className="px-2">
                            <ActionIcon
                                variant="outline" color="gray"
                                onClick={() => downloadImage(post.images[3].url, post.id)}
                            >
                                <AiOutlineDownload size="24px" />
                            </ActionIcon>
                        </Tooltip>
                        <Tooltip label="fullscreen" withArrow className="px-2">
                            <ActionIcon variant="subtle" color="gray" component="a" href={post.images[3].url}>
                                <AiOutlineArrowsAlt size="24px"></AiOutlineArrowsAlt>
                            </ActionIcon>
                        </Tooltip>
                    </ActionIcon.Group>
                </div>
            </div>
            <ImageTag post={post} similar={similar} />

            <Footer />
        </div>
    );
}
