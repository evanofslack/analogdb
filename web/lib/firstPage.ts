import type { InitialDisplay } from "@hooks/useDisplay";
import { deviceClass, displayCookie, parseDisplay } from "@lib/display";
import { ServerPostResponse } from "analogdb-generated";
import { cookies } from "next/headers";

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
      cameraMake: post.cameraMake,
      cameraModel: post.cameraModel,
      filmMake: post.filmMake,
      filmType: post.filmType,
    })),
  };
}

export async function loadDisplay(device: {
  type?: string;
}): Promise<InitialDisplay> {
  const cookieStore = await cookies();
  return {
    display: parseDisplay(cookieStore.get(displayCookie)?.value),
    width: deviceClass(device),
  };
}
