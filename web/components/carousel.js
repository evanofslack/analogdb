import styles from "./carousel.module.css";
import Image from "next/image";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Carousel } from "@mantine/carousel";
import { Progress, rem } from "@mantine/core";

export default function ImageCarousel(props) {
  let posts = props.posts;

  // we want to start carousel at index 1 to show progress
  // so shift the array so most similar is still first
  // let last = posts.pop();
  // posts.unshift(posts.pop());

  const [scrollProgress, setScrollProgress] = useState(0);
  const [embla, setEmbla] = useState(null);

  const handleScroll = useCallback(() => {
    if (!embla) return;
    const progress = Math.max(0, Math.min(1, embla.scrollProgress()));
    setScrollProgress(progress * 100);
  }, [embla, setScrollProgress]);

  useEffect(() => {
    if (embla) {
      embla.on("scroll", handleScroll);
      handleScroll();
    }
  }, [embla]);

  return (
    <div className={styles.container}>
      <h2 className={styles.title}>discover similar</h2>
      <div className={styles.carousel}>
        <Carousel
          slideSize="20%"
          slideGap="md"
          align="start"
          slidesToScroll={1}
          initialSlide={1}
          loop
          getEmblaApi={setEmbla}
          breakpoints={[
            { maxWidth: "md", slideSize: "33.3%" },
            { maxWidth: "sm", slideSize: "50%" },
          ]}
          sx={{ flex: 1 }}
        >
          {posts.map((post) => {
            return (
              <Carousel.Slide key={post.id}>
                <div key={post.id} className={styles.similarImage}>
                  <Link
                    href={`/post/${post.id}`}
                    passHref={true}
                    legacyBehavior
                  >
                    <Image
                      key={post.id}
                      priority
                      style={{ objectFit: "cover" }}
                      src={post.images[1].url}
                      alt={`image ${post.id} by ${post.author}`}
                      sizes="100vw"
                      fill
                      quality={100}
                    />
                  </Link>
                </div>
              </Carousel.Slide>
            );
          })}
        </Carousel>
        <Progress
          color="gray"
          radius="xs"
          value={scrollProgress}
          styles={{
            bar: { transitionDuration: "0ms" },
            root: { maxWidth: rem(320) },
          }}
          size="sm"
          mt="xl"
          mx="auto"
        />
      </div>
    </div>
  );
}
