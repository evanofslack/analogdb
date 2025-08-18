"use server";

import { filmsApi } from "@lib/client";
import { FilmsGetRequest, ServerFilmsResponse } from "analogdb-generated";

export async function getFilms(
  params: FilmsGetRequest
): Promise<ServerFilmsResponse> {
  try {
    const response = await filmsApi.filmsGet(params);
    return response;
  } catch (error) {
    console.error("get films api request failed:", error);
    throw error;
  }
}
