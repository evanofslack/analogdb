import { getImageProps } from "next/image";

export default function ResponsiveImage({ phoneSrc, ...props }) {
  const {
    props: { alt, ...img },
  } = getImageProps(props);
  return (
    <picture>
      {phoneSrc && phoneSrc !== img.src && (
        <source media="(max-width: 720px)" srcSet={phoneSrc} />
      )}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img alt={alt} {...img} />
    </picture>
  );
}
