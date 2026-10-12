"use client";

import Gallery from "@components/gallery";
import { ComponentProps } from "react";

export default function HomePage(props: ComponentProps<typeof Gallery>) {
  return <Gallery {...props} />;
}
