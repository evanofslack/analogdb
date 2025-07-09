import HomePage from "./home-page";
import { authorized_fetch } from "@lib/fetch";
import { Suspense } from "react";

export const metadata = {
  title: "AnalogDB",
  description: "Film photography database",
};

async function getData() {
  const numPosts = 50;
  const route = `/posts?sort=latest&page_size=${numPosts}&grayscale=false&nsfw=false`;

  const res = await authorized_fetch(route, "GET");

  if (!res.ok) {
    throw new Error("Failed to fetch data");
  }

  return res.json();
}

export default async function Page() {
  const data = await getData();
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <HomePage data={data} />
    </Suspense>
  );
}
