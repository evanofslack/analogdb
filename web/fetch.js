import { baseURL } from "./constants.js";
import getConfig from "next/config";

const { publicConfig } = getConfig();

// for buildtime
let username = process.env.AUTH_USERNAME;
let password = process.env.AUTH_PASSWORD;

// for runtime
if (username == "") {
  username = publicConfig.AUTH_USERNAME;
}
if (password == "") {
  password = publicConfig.AUTH_PASSWORD;
}

export async function authorized_fetch(route, method) {
  const url = `${baseURL}${route}`;
  let headers = new Headers();

  if (username != "" && password != "") {
    headers.set(
      "Authorization",
      "Basic " + Buffer.from(username + ":" + password).toString("base64")
    );
  }

  const response = await fetch(url, { method: method, headers: headers });
  return response;
}
