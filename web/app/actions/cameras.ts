"use server";

import { camerasApi } from "@lib/client";
import { CamerasGetRequest } from "analogdb-generated";
import { unstable_cache } from "next/cache";

export interface CameraOption {
  make: string;
  model: string;
  label: string;
}

const params: CamerasGetRequest = {
  sort: "counts",
  pageSize: 500,
  includeCounts: true,
  excludeZeroCounts: true,
};

const getCameraOptionsCached = unstable_cache(
  async (): Promise<CameraOption[]> => {
    const response = await camerasApi.camerasGet(params);
    return (response.cameras ?? [])
      .filter((c) => c.make && c.model)
      .map((c) => ({
        make: c.make,
        model: c.model,
        label: `${c.make} - ${c.model}`,
      }))
      .filter((v, i, arr) => arr.findIndex((x) => x.label === v.label) === i);
  },
  ["camera-options"],
  { revalidate: 3600 }
);

export async function getCameraOptions(): Promise<CameraOption[]> {
  try {
    return await getCameraOptionsCached();
  } catch (error) {
    console.error("get camera options request failed:", error);
    return [];
  }
}
