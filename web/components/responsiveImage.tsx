import { getImageProps, ImageProps } from "next/image";

interface ResponsiveImageProps extends ImageProps {
  phoneSrc?: string;
}

export default function ResponsiveImage({
  phoneSrc,
  ...props
}: ResponsiveImageProps) {
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
