import { baseURL } from "./constants.js";

export async function authorized_fetch(route, method = "GET") {
  const url = `${baseURL}${route}`;
  let headers = {};

  const username = process.env.AUTH_USERNAME;
  const password = process.env.AUTH_PASSWORD;

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
