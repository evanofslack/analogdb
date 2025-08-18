import {
  CamerasApi,
  Configuration,
  FilmsApi,
  PostApi,
  PostsApi,
} from "analogdb-generated";
import { baseURL } from "./constants";

const username = process.env.AUTH_USERNAME;
const password = process.env.AUTH_PASSWORD;
const auth = Buffer.from(`${username}:${password}`).toString("base64");

console.log("Creating authenticated client");
console.log("User:", process.env.AUTH_USERNAME);
console.log("Password:", process.env.AUTH_PASSWORD);

const config = new Configuration({
  basePath: baseURL,
  username: username,
  password: password,
  headers: {
    Authorization: `Basic ${auth}`, // add auth headers for all requests to bypass rate limit
  },
  middleware: [
    {
      pre: async (context) => {
        console.log(
          `Request: method=${context.init.method}, url=${context.url}`
        );
        // console.log("Request headers:", context.init.headers);
        return Promise.resolve(context);
      },
    },
  ],
});

export const postApi: PostApi = new PostApi(config);
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
