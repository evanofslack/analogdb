import "server-only";
import {
  CamerasApi,
  Configuration,
  FilmsApi,
  KeywordsApi,
  PostApi,
  PostsApi,
  SearchApi,
} from "analogdb-generated";
import pkg from "../package.json";
import { baseURL } from "./constants";

const username = process.env.API_WEB_USERNAME;
const password = process.env.API_WEB_PASSWORD;
const userAgent = `analogdb-web/${pkg.version}`;
const authHeaders: Record<string, string> =
  username && password
    ? {
        Authorization: `Basic ${Buffer.from(`${username}:${password}`).toString(
          "base64"
        )}`,
      }
    : {};

const config = new Configuration({
  basePath: baseURL,
  headers: {
    ...authHeaders,
    "User-Agent": userAgent,
  },
  middleware:
    process.env.NODE_ENV === "production"
      ? []
      : [
          {
            pre: async (context) => {
              console.log(
                `Request: method=${context.init.method}, url=${context.url}`
              );
              return Promise.resolve(context);
            },
          },
        ],
});

export const postApi: PostApi = new PostApi(config);
export const postsApi: PostsApi = new PostsApi(config);
export const filmsApi: FilmsApi = new FilmsApi(config);
export const camerasApi: CamerasApi = new CamerasApi(config);
export const searchApi: SearchApi = new SearchApi(config);
export const keywordsApi: KeywordsApi = new KeywordsApi(config);

export async function authorized_fetch(
  route: string,
  method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" = "GET",
  revalidate: number = 60
): Promise<Response> {
  const url = `${baseURL}${route}`;
  const headers: Record<string, string> = {
    ...authHeaders,
    "User-Agent": userAgent,
  };

  const response = await fetch(url, {
    method: method,
    headers: headers,
    next: { revalidate },
  } as RequestInit);

  return response;
}
