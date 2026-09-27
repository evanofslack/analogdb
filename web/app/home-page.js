"use client";

import Gallery from "@components/gallery";

export default function HomePage({ isAdmin }) {
  return <Gallery isAdmin={isAdmin} />;
}
