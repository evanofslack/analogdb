"use server";

import { camerasApi } from "@lib/client";
import { CamerasGetRequest, ServerCamerasResponse } from "analogdb-generated";

export async function getCameras(
  params: CamerasGetRequest
): Promise<ServerCamerasResponse> {
  try {
    const response = await camerasApi.camerasGet(params);
    return response;
  } catch (error) {
    console.error("get cameras api request failed:", error);
    throw error;
  }
}
