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
import { useEffect, useState } from "react";

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
  const [isHighResLoaded, setIsHighResLoaded] = useState(false);

  useEffect(() => {
    startNavigationProgress();
  }, []);

  let post = props.post;
  let lowResImage = post.images[2];
  let highResImage = post.images[3];
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
            priority
            style={
              isHighResLoaded ? { display: "none" } : { objectFit: "contain" }
            }
            fill
            src={lowResImage.url}
            alt={`image ${post.id} by ${post.author}`}
            sizes="100vw"
            quality={100}
            onLoadingComplete={completeNavigationProgress}
          />
          {/* replace with full resolution picture when loaded */}
          <Image
            priority
            style={{ objectFit: "contain" }}
            fill
            src={highResImage.url}
            alt={`image ${post.id} by ${post.author}`}
            sizes="100vw"
            quality={100}
            onLoadingComplete={() => setIsHighResLoaded(true)}
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
