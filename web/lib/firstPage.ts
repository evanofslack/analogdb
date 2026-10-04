import { ServerPostResponse } from "analogdb-generated";

export const firstPageSize = 40;

export function slim(response: ServerPostResponse): ServerPostResponse {
  return {
    meta: response.meta,
    posts: (response.posts ?? []).map((post) => ({
      id: post.id,
      author: post.author,
      title: post.title,
      caption: post.caption,
      images: (post.images ?? []).slice(0, 2),
      colors: (post.colors ?? []).slice(0, 1),
    })),
  };
}

export function guessColumns(device: { type?: string }): number {
  if (device.type === "mobile") return 2;
  if (device.type === "tablet") return 3;
  return 4;
}
