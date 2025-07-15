import { Configuration, PostsApi } from "analogdb-generated";
import { baseURL } from "./constants.js";

const username = process.env.AUTH_USERNAME;
const password = process.env.AUTH_PASSWORD;

const config = new Configuration({
  basePath: baseURL,
  username: username,
  password: password,
});

export const postsApi = new PostsApi(config);

// Keep this for other endpoints not yet converted
export async function authorized_fetch(route, method = "GET") {
  const url = `${baseURL}${route}`;
  let headers = {};

  if (username && password) {
    const auth = Buffer.from(`${username}:${password}`).toString("base64");
    headers["Authorization"] = `Basic ${auth}`;
  }

  const response = await fetch(url, {
    method: method,
    headers: headers,
    next: { revalidate: 60 },
  });

  return response;
}
