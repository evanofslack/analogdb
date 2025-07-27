import {
  CamerasApi,
  Configuration,
  FilmsApi,
  PostsApi,
} from "analogdb-generated";
import { baseURL } from "./constants";

const username = process.env.AUTH_USERNAME;
const password = process.env.AUTH_PASSWORD;

const config = new Configuration({
  basePath: baseURL,
  username: username,
  password: password,
});

export const postsApi: PostsApi = new PostsApi(config);
export const filmsApi: FilmsApi = new FilmsApi(config);
export const camerasApi: CamerasApi = new CamerasApi(config);

export async function authorized_fetch(
  route: string,
  method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" = "GET"
): Promise<Response> {
  const url = `${baseURL}${route}`;
  let headers: Record<string, string> = {};

  if (username && password) {
    const auth = Buffer.from(`${username}:${password}`).toString("base64");
    headers["Authorization"] = `Basic ${auth}`;
  }

  const response = await fetch(url, {
    method: method,
    headers: headers,
    next: { revalidate: 60 },
  } as RequestInit);

  return response;
}
