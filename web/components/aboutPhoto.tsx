"use client";

import { AboutPhoto as Photo } from "@lib/about";
import { HoverCard } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { IconAperture, IconCamera, IconMovie } from "@tabler/icons-react";
import Image from "next/image";
import Link from "next/link";
import React, { createContext, useContext } from "react";
import styles from "./aboutPhoto.module.css";

// one media query listener for the whole page, not one per photo
const CanHover = createContext(false);

export function AboutPhotoProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const canHover = useMediaQuery("(hover: hover)") ?? false;
  return <CanHover.Provider value={canHover}>{children}</CanHover.Provider>;
}

interface AboutPhotoProps {
  photo: Photo;
  small?: boolean;
  fill?: boolean;
  sizes?: string;
  className?: string;
  imageClassName?: string;
  style?: React.CSSProperties;
  children?: React.ReactNode;
  onHover?: (hovering: boolean) => void;
}

function PhotoDetails({ photo }: { photo: Photo }) {
  return (
    <div className={styles.card}>
      <span className={styles.id}>#{photo.id}</span>
      {(photo.camera || photo.lens || photo.film) && (
        <div className={styles.meta}>
          {photo.camera && (
            <span className={styles.metaItem}>
              <IconCamera size={15} className={styles.icon} />
              {photo.camera}
            </span>
          )}
          {photo.lens && (
            <span className={styles.metaItem}>
              <IconAperture size={15} className={styles.icon} />
              {photo.lens}
            </span>
          )}
          {photo.film && (
            <span className={styles.metaItem}>
              <IconMovie size={15} className={styles.icon} />
              {photo.film}
            </span>
          )}
        </div>
      )}
      {photo.keywords.length > 0 && (
        <div className={styles.keywords}>
          {photo.keywords.map((word) => (
            <span key={word} className={styles.keyword}>
              {word}
            </span>
          ))}
        </div>
      )}
      {photo.colors.length > 0 && (
        <div className={styles.colors}>
          {photo.colors.map((hex, index) => (
            <span
              key={`${hex}-${index}`}
              className={styles.swatch}
              style={{ backgroundColor: hex }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export default function AboutPhoto({
  photo,
  small = false,
  fill = false,
  sizes,
  className,
  imageClassName,
  style,
  children,
  onHover,
}: AboutPhotoProps) {
  // touch screens skip the card, a tap just opens the post
  const canHover = useContext(CanHover);
  const src = small ? photo.smallUrl : photo.url;
  const imageClass = imageClassName
    ? `${styles.image} ${imageClassName}`
    : styles.image;

  const image = fill ? (
    <Image
      src={src}
      alt={photo.alt}
      fill
      sizes={sizes}
      className={imageClass}
      style={{ objectFit: "cover" }}
    />
  ) : (
    <Image
      src={src}
      alt={photo.alt}
      width={photo.width}
      height={photo.height}
      sizes={sizes}
      className={imageClass}
    />
  );

  return (
    <HoverCard
      width={240}
      shadow="md"
      openDelay={150}
      closeDelay={100}
      position="top"
      withinPortal
      disabled={!canHover}
    >
      <HoverCard.Target>
        <Link
          href={`/post/${photo.id}`}
          prefetch={false}
          className={className ? `${styles.link} ${className}` : styles.link}
          style={style}
          onMouseEnter={onHover && (() => onHover(true))}
          onMouseLeave={onHover && (() => onHover(false))}
        >
          {image}
          {children}
        </Link>
      </HoverCard.Target>
      <HoverCard.Dropdown className={styles.dropdown}>
        <PhotoDetails photo={photo} />
      </HoverCard.Dropdown>
    </HoverCard>
  );
}
