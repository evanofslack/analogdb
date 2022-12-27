import Image from "next/image";
import Link from "next/link";
import ImageTag from "../components/imageTag";
import Footer from "../components/footer";
import styles from "./imagePage.module.css";
import { AiOutlineDownload, AiOutlineArrowsAlt } from "react-icons/ai";
import { HiArrowLeft } from "react-icons/hi";
import { ActionIcon, Tooltip } from "@mantine/core";
import {
  startNavigationProgress,
  completeNavigationProgress,
  NavigationProgress,
} from "@mantine/nprogress";
import { useEffect } from "react";

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
  useEffect(() => {
    startNavigationProgress();
  }, []);

  let post = props.post;
  let placeholder = post.images[0];
  let image = post.images[3];
  return (
    <div>
      <NavigationProgress autoReset={true} />
      <div className={styles.fullscreen}>
        <div className={styles.headerIcons}>
          <Link href="/">
            <Tooltip label="back to gallery" withArrow className="px-2">
              <ActionIcon>
                <HiArrowLeft size="2rem" />
              </ActionIcon>
            </Tooltip>
          </Link>
          <Tooltip label="fullscreen" withArrow className="px-2">
            <ActionIcon component="a" href={post.images[3].url}>
              <AiOutlineArrowsAlt size="2rem"></AiOutlineArrowsAlt>
            </ActionIcon>
          </Tooltip>
        </div>
        <div className={styles.imageContainer}>
          <Image
            fill
            src={image.url}
            alt={`image ${post.id} by ${post.author}`}
            sizes="100vw"
            style={{ objectFit: "contain" }}
            quality={100}
            placeholder="blur"
            blurDataURL={placeholder.url}
            onLoadingComplete={completeNavigationProgress}
          />
        </div>
        <div className={styles.footerIcons}>
          <Tooltip label="download" withArrow className="px-2">
            <ActionIcon
              onClick={() => downloadImage(post.images[3].url, post.id)}
            >
              <AiOutlineDownload size="2rem" />
            </ActionIcon>
          </Tooltip>
        </div>
      </div>
      <ImageTag post={post} />
      <Footer />
    </div>
  );
}
